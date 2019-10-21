// Package lockness implementes facilities for a Golang manager
// for Learning Locker
// Testing LL_API_KEY: 2c617bb5701e0a67b54252110f0ddf11672b4820
// Testing LL_API_SECRET: e1900213b5e375b3c3f3e054b1e12d8f534b8c8c
package lockness

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// Statements is a struct that can a successful Learning Locker
// API response can be decoded into.  It includes a "More" property
// to indicate that the API has more data to return, and is includes
// as slice of Statement with the return data. An example of the
// struct format can be seen in testdata/person.json.
type Statements struct {
	More       string      `json:"more"`
	Statements []Statement `json:"statements"`
}

// Statement is a struct that we send to Learning Locker
type Statement struct {
	Actor  Actor    `json:"actor"`
	Verb   Verb     `json:"verb"`
	Target Activity `json:"object"`
}

// Actor is a struct for the actor portion of the xapi statement
type Actor struct {
	Mbox string `json:"mbox"`
}

// Verb is a struct for the verb portion of the xapi statement
type Verb struct {
	ID      string      `json:"id"`
	Display VerbDisplay `json:"display"`
}

// VerbDisplay is a struct for the display portion of the verb xapi tag
type VerbDisplay struct {
	Verb string `json:"en-US"`
}

// Activity is a struct for the object portion of the xapi statement
type Activity struct {
	ID         string             `json:"id"`
	Definition ActivityDefinition `json:"definition"`
}

// ActivityDefinition is a struct for the definition portion of the object xapi tag
type ActivityDefinition struct {
	Name       ActivityName        `json:"name"`
	Desciption ActivityDescription `json:"description"`
}

// ActivityName is a struct that names the object definition in the object xapi tag
type ActivityName struct {
	ActName string `json:"en-US"`
}

// ActivityDescription is a struct that describes the object
type ActivityDescription struct {
	DesName string `json:"en-US"`
}

//LLResult stores information coming back from an API call to learning locker
type LLResult struct {
	ModuleID            string
	ModuleName          string
	CheckpointCompleted int
	NumCheckpoints      int
}

//ProgressData stores a list of learning locker results
type ProgressData struct {
	Progress []LLResult
}

//LearnerData stores a learner's progress results from learning locker
type LearnerData struct {
	FirstName string
	LastName  string
	Username  string
	Role      string
	ProgressData
}

//MentorData stores all of the learners' progress currently listed in learning locker
type MentorData struct {
	Learners []LearnerData
}

// LLRequest composes a request to get a users progress
type LLRequest struct {
	ReqString        string `yaml:"userReqString"` // format string for the http request
	PostString       string `yaml:"llPostString"`  // format string for the Post request
	LearningLockerIP string `yaml:"llIP"`          // Learning Locker IP address
	APIVersion       string `yaml:"llAPIVersion"`  // xAPI version
	LLApiKey         string ``                     // Learning Locker Auth Key
	LLSecretKey      string ``                     // Learning Locker Secret Key
	Err              error  ``                     // returns non-nil for any constructor problem
}

// ModuleInfo stores module info with number of total checkpoints
type ModuleInfo struct {
	ModuleID    string `json:"moduleID"`
	ModuleName  string `json:"moduleName"`
	TotalChecks int    `json:"totalCheckPoints"`
}

// JSONDB implements the Database interface for a JSON file
// storage of module information.
type JSONDB struct {
	moduleFile string
	moduleInfo []ModuleInfo
	Err        error
}

// NewLLRequest return a new *LLRequest.  Users should verify that
// LLRequest.Err is nil before using the return pointer.
func NewLLRequest(config string, modules string) (*LLRequest, *JSONDB) {

	llReq := getRequest(config)
	modulesDB := getModules(modules)

	if modulesDB.Err != nil {
		log.Fatalf("unable to connect to module information %s", modulesDB.Err)
	}

	return &llReq, &modulesDB
}

func getRequest(config string) LLRequest {

	var llReq LLRequest

	yamlFile, err := ioutil.ReadFile(config)
	if err != nil {
		llReq.Err = fmt.Errorf("unable to open the learning locker config yaml file: %s", err)
	}

	err = yaml.Unmarshal(yamlFile, &llReq)
	if err != nil {
		llReq.Err = fmt.Errorf("unable to unmarshal the learning locker config file: %s", err)
	}

	if key := os.Getenv("LL_API_KEY"); key == "" {
		llReq.Err = fmt.Errorf("missing environment variable: LL_API_KEY")
	}

	if secret := os.Getenv("LL_API_SECRET"); secret == "" {
		llReq.Err = fmt.Errorf("missing environment variable: LL_API_SECRET")
	}

	llReq.LLApiKey = os.Getenv("LL_API_KEY")
	llReq.LLSecretKey = os.Getenv("LL_API_SECRET")

	return llReq
}

func getModules(modules string) JSONDB {

	var db JSONDB
	db.moduleFile = modules
	f, err := os.Open(db.moduleFile)
	defer f.Close()
	if err != nil {
		db.Err = fmt.Errorf("unable to open modulesFile: %s", err)
	}
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	err = dec.Decode(&db.moduleInfo)
	if err != nil {
		db.Err = fmt.Errorf("%s: unable to decode into ModuleInfo: %s", db.Err, err)
	}

	return db
}

// ProgressURL returns the Learning Locker url for retrieving data for specific user
func (llr *LLRequest) ProgressURL(username string) string {
	url := fmt.Sprintf(llr.ReqString, llr.LearningLockerIP, username)
	return url
}

// Progress requests Learning Locker data for specific username. The API call will return the first 100 statements,
// If there are more than 100, the json response will have a 'more' link that contains the next 100.
// The function will loop everytime there is a 'more' link, make a new API call and combine all the data.
// The function will stop looping when 'more' contains no link.
func (llr *LLRequest) Progress(username string, db *JSONDB) (ProgressData, error) {

	var statements Statements
	pd := ProgressData{} // Blank ProgressData to return with errors
	notDone := true
	currentEndpoint := llr.ProgressURL(username)

	for notDone {

		client := &http.Client{}
		req, err := http.NewRequest("GET", currentEndpoint, nil)
		if err != nil {
			return pd, err
		}

		req.Header.Add("Authorization", "Basic "+basicAuth(llr.LLApiKey, llr.LLSecretKey))
		req.Header.Add("X-Experience-API-Version", llr.APIVersion)
		response, err := client.Do(req)
		if err != nil {
			return pd, err
		}

		var byteValue []byte
		if response.StatusCode == http.StatusOK {
			byteValue, err = ioutil.ReadAll(response.Body)
			if err != nil {
				return pd, err
			}
		}

		err = json.Unmarshal(byteValue, &statements)
		if err != nil {
			return pd, err
		}

		pd, err = parseProgress(statements, pd)
		if err != nil {
			return pd, err
		}
		currentEndpoint = "http://" + llr.LearningLockerIP + statements.More

		if statements.More == "" {
			notDone = false
		}

	}

	pd.Progress = removeDuplicates(pd.Progress)
	pd.Progress = getCount(pd.Progress)
	pd.Progress = fillBlanks(pd.Progress, *db)
	return pd, nil

}

// MentorURL returns the Learning Locker url for retrieving data for all users
func (llr *LLRequest) MentorURL() string {
	url := fmt.Sprintf(llr.PostString, llr.LearningLockerIP)
	return url
}

// Mentor requests all Learning Locker data. The API call will return the first 100 statements,
// If there are more than 100, the json response will have a 'more' link that contains the next 100.
// The function will loop everytime there is a 'more' link, make a new API call and combine all the data.
// The function will stop looping when 'more' contains no link.
func (llr *LLRequest) Mentor(db *JSONDB) (MentorData, error) {

	var statements Statements
	var users = make(map[string]bool)
	md := MentorData{}
	var mentorMap = make(map[string][]LLResult)
	notDone := true
	currentEndpoint := llr.MentorURL()

	for notDone {

		client := &http.Client{}
		req, err := http.NewRequest("GET", currentEndpoint, nil)
		if err != nil {
			return md, err
		}

		req.Header.Add("Authorization", "Basic "+basicAuth(llr.LLApiKey, llr.LLSecretKey))
		req.Header.Add("X-Experience-API-Version", llr.APIVersion)
		response, err := client.Do(req)
		if err != nil {
			return md, err
		}

		var byteValue []byte
		if response.StatusCode == http.StatusOK {
			byteValue, err = ioutil.ReadAll(response.Body)
			if err != nil {
				return md, err
			}
		}

		err = json.Unmarshal(byteValue, &statements)
		if err != nil {
			return md, err
		}

		mentorMap, users, err = parseMentor(statements, users, mentorMap)
		if err != nil {
			return md, err
		}
		currentEndpoint = "http://" + llr.LearningLockerIP + statements.More

		if statements.More == "" {
			notDone = false
		}

	}

	var blank []LLResult
	for user := range users {
		_, ok := mentorMap[user]
		if !ok {
			mentorMap[user] = blank
		}
	}

	for user, result := range mentorMap {

		var lenData LearnerData

		lenData.Username = user
		lenData.ProgressData.Progress = removeDuplicates(result)
		lenData.ProgressData.Progress = getCount(lenData.ProgressData.Progress)
		lenData.ProgressData.Progress = fillBlanks(lenData.ProgressData.Progress, *db)
		md.Learners = append(md.Learners, lenData)

	}

	return md, nil

}

// basicAuth combines LL API KEY and SECRET KEY and returns base64 encoded string
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// parseProgress parses statements for LLResult data and return list of all LLResults.
func parseProgress(statements Statements, proData ProgressData) (ProgressData, error) {

	for _, api := range statements.Statements {

		var lldata LLResult
		var moduleInfo, checkPointInfo string

		moduleInfo = api.Target.Definition.Desciption.DesName
		if moduleInfo == "" {
			continue
		}
		checkPointInfo = api.Target.Definition.Name.ActName
		if checkPointInfo == "" {
			return proData, fmt.Errorf("Error parsing for Activity Description")
		}

		s1 := strings.Split(moduleInfo, "--")
		s2 := strings.Split(checkPointInfo, "--")

		if (len(s1) != 2) || (len(s2) != 2) {
			continue
		}

		lldata.ModuleID, lldata.ModuleName = s1[0], s1[1]
		lldata.CheckpointCompleted, _ = strconv.Atoi(s2[0])
		lldata.NumCheckpoints, _ = strconv.Atoi(s2[1])

		proData.Progress = append(proData.Progress, lldata)

	}

	return proData, nil

}

// parseMentor parses statements for LLResult data. The data is then appended to specific lists
// by each user. A map slice is return with all the data separated by users.
func parseMentor(statements Statements, users map[string]bool, mentorMap map[string][]LLResult) (map[string][]LLResult, map[string]bool, error) {

	var username string

	for _, api := range statements.Statements {

		var lldata LLResult
		var moduleInfo, checkPointInfo string

		username = strings.Split(strings.Split(api.Actor.Mbox, ":")[1], "@")[0]
		if !users[username] {
			users[username] = true
		}

		moduleInfo = api.Target.Definition.Desciption.DesName
		if moduleInfo == "" {
			continue
		}
		checkPointInfo = api.Target.Definition.Name.ActName
		if checkPointInfo == "" {
			return mentorMap, users, fmt.Errorf("Error parsing for Activity Description")
		}

		s1 := strings.Split(moduleInfo, "--")
		s2 := strings.Split(checkPointInfo, "--")

		if (len(s1) != 2) || (len(s2) != 2) {
			continue
		}

		lldata.ModuleID, lldata.ModuleName = s1[0], s1[1]
		lldata.CheckpointCompleted, _ = strconv.Atoi(s2[0])
		lldata.NumCheckpoints, _ = strconv.Atoi(s2[1])

		mentorMap[username] = append(mentorMap[username], lldata)

	}

	return mentorMap, users, nil

}

func removeDuplicates(pd []LLResult) []LLResult {

	seen := make(map[LLResult]bool)
	var new []LLResult

	for _, result := range pd {
		if seen[result] == false {
			seen[result] = true
			new = append(new, result)
		}
	}
	return new
}

func getCount(pd []LLResult) []LLResult {
	names := make(map[string]string)
	counts := make(map[string]int)
	totals := make(map[string]int)
	var finalPD []LLResult

	for _, result := range pd {
		counts[result.ModuleID]++
		totals[result.ModuleID] = result.NumCheckpoints
		names[result.ModuleID] = result.ModuleName
	}

	for key, val := range counts {
		temp := LLResult{}
		temp.ModuleName = names[key]
		temp.ModuleID = key
		temp.CheckpointCompleted = val
		temp.NumCheckpoints = totals[key]
		finalPD = append(finalPD, temp)
	}

	return finalPD
}

func fillBlanks(pd []LLResult, db JSONDB) []LLResult {

	ids := make(map[string]bool)

	for _, result := range pd {
		ids[result.ModuleID] = true
	}

	for _, module := range db.moduleInfo {
		if !ids[module.ModuleID] {
			var res LLResult
			res.ModuleID = module.ModuleID
			res.ModuleName = module.ModuleName
			res.CheckpointCompleted = 0
			res.NumCheckpoints = module.TotalChecks
			pd = append(pd, res)
		}
	}

	return pd
}
