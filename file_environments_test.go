package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/v2/feeds"
	"github.com/stretchr/testify/assert"
)

const (
	validEnvConfig = `
		[
			{
				"name":"test-env",
				"read-url": "https://test-env.ft.com",
				"s3-url": "http://test.s3.amazonaws.com"
			}
		]`
	validEnvCredentialsConfig = `
		[
			{
				"env-name": "test-env",
				"username": "test-user",
				"password": "test-pwd"
			}
		]`
	validValidationCredentialsConfig = `
		{
			"username": "test-user",
			"password": "test-pwd"
		}`
	invalidJsonConfig = `invalid-config`
)

func TestParseEnvsIntoMap(t *testing.T) {
	envsToBeParsed := getValidEnvs()
	credentials := getValidCredentials()
	environments = newThreadSafeEnvironments()

	removedEnvs := parseEnvsIntoMap(envsToBeParsed, credentials)

	assert.Equal(t, 0, len(removedEnvs))
	assert.Equal(t, len(envsToBeParsed), environments.len())
	envName := envsToBeParsed[1].Name
	assert.Equal(t, envName, environments.environment(envName).Name)
	assert.Equal(t, credentials[1].Username, environments.environment(envName).Username)
}

func TestParseEnvsIntoMapWithRemovedEnv(t *testing.T) {
	envsToBeParsed := getValidEnvs()
	credentials := getValidCredentials()
	environments = newThreadSafeEnvironments()
	environments.envMap["removed-env"] = Environment{}

	removedEnvs := parseEnvsIntoMap(envsToBeParsed, credentials)

	assert.Equal(t, 1, len(removedEnvs))
	assert.Equal(t, len(envsToBeParsed), environments.len())
	envName := envsToBeParsed[1].Name
	assert.Equal(t, envName, environments.environment(envName).Name)
	assert.Equal(t, credentials[1].Username, environments.environment(envName).Username)
}

func TestParseEnvsIntoMapWithExistingEnv(t *testing.T) {
	envsToBeParsed := getValidEnvs()
	credentials := getValidCredentials()
	environments = newThreadSafeEnvironments()
	existingEnv := envsToBeParsed[0]
	environments.envMap[existingEnv.Name] = existingEnv

	removedEnvs := parseEnvsIntoMap(envsToBeParsed, credentials)

	assert.Equal(t, 0, len(removedEnvs))
	assert.Equal(t, len(envsToBeParsed), environments.len())
	envName := envsToBeParsed[1].Name
	assert.Equal(t, envName, environments.environment(envName).Name)
	assert.Equal(t, credentials[1].Username, environments.environment(envName).Username)
}

func TestParseEnvsIntoMapWithNoCredentials(t *testing.T) {
	envsToBeParsed := getValidEnvs()
	credentials := []Credentials{}
	environments = newThreadSafeEnvironments()

	removedEnvs := parseEnvsIntoMap(envsToBeParsed, credentials)

	assert.Equal(t, 0, len(removedEnvs))
	assert.Equal(t, len(envsToBeParsed), environments.len())
	envName := envsToBeParsed[1].Name
	assert.Equal(t, envName, environments.environment(envName).Name)
}

func TestFilterInvalidEnvs(t *testing.T) {
	envsToBeFiltered := getValidEnvs()

	filteredEnvs := filterInvalidEnvs(envsToBeFiltered)

	assert.Equal(t, len(envsToBeFiltered), len(filteredEnvs))
}

func TestFilterInvalidEnvsWithEmptyName(t *testing.T) {
	envsToBeFiltered := []Environment{
		{
			Name:     "",
			ReadUrl:  "test",
			S3Url:    "test",
			Username: "dummy",
			Password: "dummy",
		},
	}

	filteredEnvs := filterInvalidEnvs(envsToBeFiltered)

	assert.Equal(t, 0, len(filteredEnvs))
}

func TestFilterInvalidEnvsWithEmptyReadUrl(t *testing.T) {
	envsToBeFiltered := []Environment{
		{
			Name:     "test",
			ReadUrl:  "",
			S3Url:    "test",
			Username: "dummy",
			Password: "dummy",
		},
	}

	filteredEnvs := filterInvalidEnvs(envsToBeFiltered)

	assert.Equal(t, 0, len(filteredEnvs))
}

func TestFilterInvalidEnvsWithEmptyS3Url(t *testing.T) {
	envsToBeFiltered := []Environment{
		{
			Name:     "test",
			ReadUrl:  "test",
			S3Url:    "",
			Username: "dummy",
			Password: "dummy",
		},
	}

	filteredEnvs := filterInvalidEnvs(envsToBeFiltered)

	assert.Equal(t, 1, len(filteredEnvs))
}

func TestFilterInvalidEnvsWithEmptyUsernameUrl(t *testing.T) {
	envsToBeFiltered := []Environment{
		{
			Name:     "test",
			ReadUrl:  "test",
			S3Url:    "test",
			Username: "",
			Password: "dummy",
		},
	}

	filteredEnvs := filterInvalidEnvs(envsToBeFiltered)

	assert.Equal(t, 1, len(filteredEnvs))
}

func TestFilterInvalidEnvsWithEmptyPwd(t *testing.T) {
	envsToBeFiltered := []Environment{
		{
			Name:     "test",
			ReadUrl:  "test",
			S3Url:    "test",
			Username: "test",
			Password: "",
		},
	}

	filteredEnvs := filterInvalidEnvs(envsToBeFiltered)

	assert.Equal(t, 1, len(filteredEnvs))
}

func TestUpdateValidationCredentialsHappyFlow(t *testing.T) {
	fileName := prepareFile(validValidationCredentialsConfig)
	fileContents, _ := ioutil.ReadFile(fileName)
	err := updateValidationCredentials(fileContents)

	assert.Nil(t, err)
	assert.Equal(t, "test-user:test-pwd", validatorCredentials)
	os.Remove(fileName)
}

func TestUpdateValidationCredentialNilFile(t *testing.T) {
	validatorCredentials := Credentials{
		Username: "test-username",
		Password: "test-password",
	}
	err := updateValidationCredentials(nil)

	assert.NotNil(t, err)
	//make sure validationCredentials didn't change after failing call to updateValidationCredentials().
	assert.Equal(t, "test-username", validatorCredentials.Username)
	assert.Equal(t, "test-password", validatorCredentials.Password)
}

func TestUpdateValidationCredentialsInvalidConfig(t *testing.T) {
	fileName := prepareFile(invalidJsonConfig)
	validatorCredentials := Credentials{
		Username: "test-username",
		Password: "test-password",
	}
	fileContents, _ := ioutil.ReadFile(fileName)
	err := updateValidationCredentials(fileContents)
	assert.NotNil(t, err)
	//make sure validationCredentials didn't change after failing call to updateValidationCredentials().
	assert.Equal(t, "test-username", validatorCredentials.Username)
	assert.Equal(t, "test-password", validatorCredentials.Password)
	os.Remove(fileName)
}

func TestConfigureFeedsWithEmptyListOfMetrics(t *testing.T) {
	subscribedFeeds["test-feed"] = []feeds.Feed{
		MockFeed{},
	}
	appConfig = &AppConfig{}

	configureFileFeeds(make(map[string]Environment), []string{"test-feed"})

	assert.Equal(t, 0, len(subscribedFeeds))
}

func TestUpdateEnvsHappyFlow(t *testing.T) {
	subscribedFeeds["test-feed"] = []feeds.Feed{
		MockFeed{},
	}
	appConfig = &AppConfig{}
	envsFileName := prepareFile(validEnvConfig)
	envsFileContents, _ := ioutil.ReadFile(envsFileName)

	envCredsFileName := prepareFile(validEnvCredentialsConfig)
	credsFileContents, _ := ioutil.ReadFile(envCredsFileName)

	err := updateEnvs(envsFileContents, credsFileContents)

	assert.Nil(t, err)
	os.Remove(envsFileName)
	os.Remove(envCredsFileName)
}

func TestUpdateEnvsHappyNilEnvsFile(t *testing.T) {
	envCredsFileName := prepareFile(validEnvCredentialsConfig)
	credsFileContents, _ := ioutil.ReadFile(envCredsFileName)
	err := updateEnvs(nil, credsFileContents)

	assert.NotNil(t, err)
	os.Remove(envCredsFileName)
}

func TestUpdateEnvsNilEnvCredentialsFile(t *testing.T) {
	envsFileName := prepareFile(validEnvConfig)
	envsFileContents, _ := ioutil.ReadFile(envsFileName)

	err := updateEnvs(envsFileContents, nil)

	assert.NotNil(t, err)
	os.Remove(envsFileName)
}

func TestComputeMD5Hash(t *testing.T) {
	var testCases = []struct {
		caseDescription string
		toHash          []byte
		expectedHash    string
	}{
		{
			caseDescription: "one-line valid input",
			toHash:          []byte("foobar"),
			expectedHash:    "3858f62230ac3c915f300c664312c63f",
		},
		{
			caseDescription: "multi-line valid input",
			toHash: []byte(`foo
					      bar`),
			expectedHash: "1be7783a9859a16a010d466d39342543",
		},
		{
			caseDescription: "empty input",
			toHash:          []byte(""),
			expectedHash:    "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			caseDescription: "nil input",
			toHash:          nil,
			expectedHash:    "d41d8cd98f00b204e9800998ecf8427e",
		},
	}

	for _, tc := range testCases {
		actualHash, err := computeMD5Hash(tc.toHash)
		assert.Nil(t, err)
		assert.Equal(t, tc.expectedHash, actualHash,
			fmt.Sprintf("%s: Computed has doesn't match expected hash", tc.caseDescription))
	}
}

func TestIsFileChanged(t *testing.T) {
	var testCases = []struct {
		caseDescription       string
		fileContents          []byte
		fileName              string
		configFilesHashValues map[string]string
		expectedResult        bool
		expectedHash          string
	}{
		{
			caseDescription: "file not changed",
			fileContents:    []byte("foobar"),
			fileName:        "file1",
			configFilesHashValues: map[string]string{
				"file1": "3858f62230ac3c915f300c664312c63f",
				"file2": "1be7783a9859a16a010d466d39342543",
			},
			expectedResult: false,
			expectedHash:   "3858f62230ac3c915f300c664312c63f",
		},
		{
			caseDescription: "new file",
			fileContents:    []byte("foobar"),
			fileName:        "file1",
			configFilesHashValues: map[string]string{
				"file2": "1be7783a9859a16a010d466d39342543",
			},
			expectedResult: true,
			expectedHash:   "3858f62230ac3c915f300c664312c63f",
		},
		{
			caseDescription: "file contents changed",
			fileContents:    []byte("foobarNew"),
			fileName:        "file1",
			configFilesHashValues: map[string]string{
				"file1": "3858f62230ac3c915f300c664312c63f",
				"file2": "1be7783a9859a16a010d466d39342543",
			},
			expectedResult: true,
			expectedHash:   "bdcf75c01270b40ebb33c1d24457ed81",
		},
	}

	for _, tc := range testCases {
		configFilesHashValues = tc.configFilesHashValues
		actualResult, actualHash, _ := isFileChanged(tc.fileContents, tc.fileName)
		assert.Equal(t, tc.expectedResult, actualResult,
			fmt.Sprintf("%s: File change was not detected correctly.", tc.caseDescription))
		assert.Equal(t, tc.expectedHash, actualHash,
			fmt.Sprintf("%s: The expected file hash was not returned.", tc.caseDescription))
	}
}

func TestUpdateEnvsIfChangedEnvFileDoesntExist(t *testing.T) {
	credsFile := prepareFile(validEnvCredentialsConfig)
	defer os.Remove(credsFile)

	environments = newThreadSafeEnvironments()
	configFilesHashValues = make(map[string]string)

	err := updateEnvsIfChanged("thisFileDoesntexist", credsFile)

	assert.NotNil(t, err, "Didn't get an error after supplying file which doesn't exist")
	assert.Equal(t, 0, environments.len(), "No new environments should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No hashes should've been updated")
}

func TestUpdateEnvsIfChangedCredsFileDoesntExist(t *testing.T) {
	envsFile := prepareFile(validEnvConfig)
	defer os.Remove(envsFile)

	environments = newThreadSafeEnvironments()
	configFilesHashValues = make(map[string]string)

	err := updateEnvsIfChanged(envsFile, "thisFileDoesntexist")

	assert.NotNil(t, err, "Didn't get an error after supplying file which doesn't exist")
	assert.Equal(t, 0, environments.len(), "No new environments should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No hashes should've been updated")
}

func TestUpdateEnvsIfChangedFilesDontExist(t *testing.T) {
	environments = newThreadSafeEnvironments()
	configFilesHashValues = make(map[string]string)

	err := updateEnvsIfChanged("thisFileDoesntExist", "thisDoesntExistEither")

	assert.NotNil(t, err, "Didn't get an error after supplying files which don't exist")
	assert.Equal(t, 0, environments.len(), "No new environments should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No hashes should've been updated")
}

func TestUpdateEnvsIfChangedValidFiles(t *testing.T) {
	envsFile := prepareFile(validEnvConfig)
	defer os.Remove(envsFile)
	credsFile := prepareFile(validEnvCredentialsConfig)
	defer os.Remove(credsFile)

	environments = newThreadSafeEnvironments()
	configFilesHashValues = make(map[string]string)

	//appConfig has to be non-nil for the actual update to work
	appConfig = &AppConfig{}
	err := updateEnvsIfChanged(envsFile, credsFile)

	assert.Nil(t, err, "Got an error after supplying valid files")
	assert.Equal(t, 1, environments.len(), "New environment should've been added")
	assert.Equal(t, 2, len(configFilesHashValues), "New hashes should've been added")
}

func TestUpdateEnvsIfChangedNoChanges(t *testing.T) {
	envsFile := prepareFile(validEnvConfig)
	defer os.Remove(envsFile)
	credsFile := prepareFile(validEnvCredentialsConfig)
	defer os.Remove(credsFile)

	environments = newThreadSafeEnvironments()
	environments.envMap = map[string]Environment{
		"test-env": {
			Name:     "test-env",
			Password: "test-pwd",
			ReadUrl:  "https://test-env.ft.com",
			S3Url:    "http://test.s3.amazonaws.com",
			Username: "test-user",
		},
	}
	configFilesHashValues = map[string]string{
		envsFile:  "792c5a9eebad1a967faab8defd9e646b",
		credsFile: "dfd8aecc21b7017c5e4f171e3279fc68",
	}

	//if the update works (which it shouldn't) we will have a failure
	appConfig = nil
	err := updateEnvsIfChanged(envsFile, credsFile)

	assert.Nil(t, err, "Got an error after supplying valid files")
	assert.Equal(t, 1, environments.len(), "Environments shouldn't have changed")
	assert.Equal(t, 2, len(configFilesHashValues), "Hashes shouldn't have changed")
}

func TestUpdateEnvsIfChangedInvalidEnvsFile(t *testing.T) {
	envsFile := prepareFile(invalidJsonConfig)
	defer os.Remove(envsFile)
	credsFile := prepareFile(validEnvCredentialsConfig)
	defer os.Remove(credsFile)

	environments = newThreadSafeEnvironments()
	configFilesHashValues = make(map[string]string)

	err := updateEnvsIfChanged(envsFile, credsFile)

	assert.NotNil(t, err, "Didn't get an error after supplying invalid file")
	assert.Equal(t, 0, environments.len(), "No new environment should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No new hashes should've been added")
}

func TestUpdateEnvsIfChangedInvalidCredsFile(t *testing.T) {
	envsFile := prepareFile(validEnvConfig)
	defer os.Remove(envsFile)
	credsFile := prepareFile(invalidJsonConfig)
	defer os.Remove(credsFile)

	environments = newThreadSafeEnvironments()
	configFilesHashValues = make(map[string]string)

	err := updateEnvsIfChanged(envsFile, credsFile)

	assert.NotNil(t, err, "Didn't get an error after supplying invalid file")
	assert.Equal(t, 0, environments.len(), "No new environment should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No new hashes should've been added")
}

func TestUpdateEnvsIfChangedInvalidFiles(t *testing.T) {
	envsFile := prepareFile(invalidJsonConfig)
	defer os.Remove(envsFile)
	credsFile := prepareFile(invalidJsonConfig)
	defer os.Remove(credsFile)

	environments = newThreadSafeEnvironments()
	configFilesHashValues = make(map[string]string)

	err := updateEnvsIfChanged(envsFile, credsFile)

	assert.NotNil(t, err, "Didn't get an error after supplying invalid file")
	assert.Equal(t, 0, environments.len(), "No new environment should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No new hashes should've been added")
}

func TestUpdateValidationCredentialsIfChangedFileDoesntExist(t *testing.T) {
	validatorCredentials = ""
	configFilesHashValues = make(map[string]string)

	err := updateValidationCredentialsIfChanged("thisFileDoesntExist")

	assert.NotNil(t, err, "Didn't get an error after supplying file which doesn't exist")
	assert.Equal(t, 0, len(validatorCredentials), "No validator credentials should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No hashes should've been updated")
}

func TestUpdateValidationCredentialsIfChangedInvalidFile(t *testing.T) {
	validationCredsFile := prepareFile(invalidJsonConfig)
	defer os.Remove(validationCredsFile)

	validatorCredentials = ""
	configFilesHashValues = make(map[string]string)

	err := updateValidationCredentialsIfChanged(validationCredsFile)

	assert.NotNil(t, err, "Didn't get an error after supplying file which doesn't exist")
	assert.Equal(t, 0, len(validatorCredentials), "No validator credentials should've been added")
	assert.Equal(t, 0, len(configFilesHashValues), "No hashes should've been updated")
}

func TestUpdateValidationCredentialsIfChangedNewFile(t *testing.T) {
	validationCredsFile := prepareFile(validValidationCredentialsConfig)
	defer os.Remove(validationCredsFile)

	validatorCredentials = ""
	configFilesHashValues = make(map[string]string)

	err := updateValidationCredentialsIfChanged(validationCredsFile)

	assert.Nil(t, err, "Shouldn't get an error for valid file")
	assert.Equal(t, "test-user:test-pwd", validatorCredentials, "New validator credentials should've been added")
	assert.Equal(t, 1, len(configFilesHashValues), "New hashes should've been added")
}

func TestUpdateValidationCredentialsIfChangedFileUnchanged(t *testing.T) {
	validationCredsFile := prepareFile(validValidationCredentialsConfig)
	defer os.Remove(validationCredsFile)

	validatorCredentials = "test-user:test-pwd"
	configFilesHashValues = map[string]string{
		validationCredsFile: "cc4d51dfe137ec8cbba8fd3ff24474be",
	}

	err := updateValidationCredentialsIfChanged(validationCredsFile)
	assert.Nil(t, err, "Shouldn't get an error for valid file")
	assert.Equal(t, "test-user:test-pwd", validatorCredentials, "Validator credentials shouldn't have changed")
	assert.Equal(t, "cc4d51dfe137ec8cbba8fd3ff24474be", configFilesHashValues[validationCredsFile], "Hashes shouldn't have changed")
}

func TestTickerWithInitialDelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	delay := 2
	ticker := newTicker(time.Duration(delay)*time.Second, time.Minute)
	defer ticker.Stop()

	before := time.Now()
	go func() {
		<-ticker.C
		cancel()
	}()

	select {
	case <-ctx.Done():
		assert.WithinDuration(t, before.Add(time.Duration(delay)*time.Second), time.Now(), time.Second, "initial tick")
	case <-time.After(time.Duration(delay+1) * time.Second):
		assert.Fail(t, "timed out waiting for initial tick")
	}
}

func prepareFile(fileContent string) string {
	file, err := ioutil.TempFile(os.TempDir(), "")
	if err != nil {
		panic("Cannot create temp file.")
	}

	writer := bufio.NewWriter(file)
	defer file.Close()
	fmt.Fprintln(writer, fileContent)
	writer.Flush()
	return file.Name()
}

func getValidEnvs() []Environment {
	return []Environment{
		{
			Name:    "test",
			ReadUrl: "test-url",
			S3Url:   "test-s3-url",
		},
		{
			Name:    "test2",
			ReadUrl: "test-url2",
			S3Url:   "test-s3-url2",
		},
	}
}

func getValidCredentials() []Credentials {
	return []Credentials{
		{
			EnvName:  "test",
			Username: "dummy-user",
			Password: "dummy-pwd",
		},
		{
			EnvName:  "test2",
			Username: "dummy-user2",
			Password: "dummy-pwd2",
		},
	}
}

type MockFeed struct{}

func (f MockFeed) Start() {}
func (f MockFeed) Stop()  {}
func (f MockFeed) FeedName() string {
	return ""
}
func (f MockFeed) FeedURL() string {
	return ""
}
func (f MockFeed) FeedType() string {
	return ""
}
func (f MockFeed) SetCredentials(username string, password string) {}
func (f MockFeed) NotificationsFor(uuid string) []*feeds.Notification {
	return nil
}
