package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type Bouncer struct {
	membersIdentifiers  []string
	allowedJidsFilePath string
}

func NewBouncer(allowdMembersFilePath string) *Bouncer {

	bouncer := &Bouncer{
		allowedJidsFilePath: allowdMembersFilePath,
		membersIdentifiers:  parseAllowedJids(allowdMembersFilePath),
	}

	return bouncer
}

func (bouncer *Bouncer) SetAllowedMembers(overideMembersIdentifiers []string) {
	bouncer.membersIdentifiers = overideMembersIdentifiers

	if err := saveAllowJids(bouncer.allowedJidsFilePath, bouncer.membersIdentifiers); err != nil {
		log.Default().Println("could not save allowed jids: ", err)
	}
}

func (bouncer *Bouncer) GetAllowedMembers() []string {
	log.Default().Println("allowed", bouncer.membersIdentifiers)
	return bouncer.membersIdentifiers
}

func (bouncer *Bouncer) isAllowd(identifier string) bool {
	for _, memberIdentifier := range bouncer.membersIdentifiers {
		if memberIdentifier == identifier {
			return true
		}
	}

	return false
}

func parseAllowedJids(allowedJidsPath string) []string {
	jsonFile, err := os.Open(allowedJidsPath)
	if err != nil {
		log.Fatalf("Failed to open JSON file: %s", err)
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf("Failed to read JSON file: %s", err)
	}

	var allowed_jids []string
	err = json.Unmarshal(byteValue, &allowed_jids)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSON data: %s", err)
	}

	return allowed_jids
}

func saveAllowJids(allowedJidsPath string, allowedJids []string) error {
	content, err := json.Marshal(allowedJids)
	if err != nil {
		return err
	}

	err = os.WriteFile(allowedJidsPath, content, 0644)
	if err != nil {
		return err
	}

	return nil
}
