package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

// ---------------------- Types

////////////////////////////////////////////////////
// Represents an entry in the ldap tree
type ldapEntry struct {
	dn         string
	attributes map[string][]string
}

func (e ldapEntry) serializeDN() string {
	return fmt.Sprintf("dn: %s\n", e.dn)
}

func (e ldapEntry) serializeAttributes() string {
	var builder strings.Builder
	for name, values := range e.attributes {
		for _, value := range values {
			builder.WriteString(fmt.Sprintf("%s: %s\n", name, value))
		}
	}
	return builder.String()
}

func (e ldapEntry) serialize() string {
	return e.serializeDN() + e.serializeAttributes()
}

// Gets the ldapentry attributes, as a string slice, sorted
func (e ldapEntry) getSortedAttributes() []string {
	attributesAsStrings := make([]string, 0)
	for k, vv := range e.attributes {
		for _, v := range vv {
			attributesAsStrings = append(attributesAsStrings, fmt.Sprintf("%s: %s", k, v))
		}
	}
	sort.Strings(attributesAsStrings)

	return attributesAsStrings
}

/////////////////////////////////////////////////////
type ldapEntryList []ldapEntry

// Methods of the Interface interface for sorting based on dn
func (l ldapEntryList) Len() int {
	return len(l)
}

func (l ldapEntryList) Less(i, j int) bool {
	return l[i].dn < l[j].dn
}

func (l ldapEntryList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// Helper implementing the stringer interface
func (l ldapEntryList) String() string {
	var builder strings.Builder
	for _, entry := range l {
		builder.WriteString(fmt.Sprintf("dn: %s\n", entry.dn))
		for _, nv := range entry.getSortedAttributes() {
			builder.WriteString(fmt.Sprintln(nv))
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

// ---------------------- End types

var currentConfigFilePtr = flag.String("current", "", "File with current configuration. Mandatory")
var newConfigFilePtr = flag.String("new", "", "File with configuration to apply. Mandatory")
var isDebug = flag.Bool("debug", false, "Writes tracing information in stdout")
var help = flag.Bool("help", false, "Shows help")

/*
Takes as an input two files with ldif format (current and new), compares them and generates as
standard output the commands to use in ldapmodify to change from current to new.
*/
func main() {

	// Treat command line parameters
	flag.Parse()

	if *help == true {
		fmt.Println("ldifCompare: Generates a ldapmodify file from moving from current to new configuration")
		flag.VisitAll(func(flag *flag.Flag) {
			fmt.Printf("\t-%s: %s\n", flag.Name, flag.Usage)
		})
		return
	}

	if *currentConfigFilePtr == "" {
		fmt.Println("[ERROR] current config file not specified")
		return
	}

	if *newConfigFilePtr == "" {
		fmt.Println("[ERROR] new config file not specified")
		return
	}

	// Read input files
	currentFileBytes, e := ioutil.ReadFile(*currentConfigFilePtr)
	if e != nil {
		fmt.Println("[ERROR] Could not read input file ", *currentConfigFilePtr)
		os.Exit(1)
	}
	newFileBytes, e := ioutil.ReadFile(*currentConfigFilePtr)
	if e != nil {
		fmt.Println("[ERROR] Could not read input file ", *currentConfigFilePtr)
		os.Exit(1)
	}

	// Read input ldiff
	currentLdapEntries := parseLdif(string(currentFileBytes))

	// Read new ldiff
	newLdapEntries := parseLdif(string(newFileBytes))

	// For debugging. Print contents of current file
	if *isDebug {
		fmt.Println("==== Current ==========================================")
		fmt.Println(currentLdapEntries)
		fmt.Print("=======================================================\n\n")
		fmt.Println("==== New  =============================================")
		fmt.Println(newLdapEntries)
		fmt.Print("=======================================================\n\n")
	}
}

// Helper function to read a config file and generate an ldapEntryList
// The results are ordered by dn and also inside each entry, by attribute
func parseLdif(ldif string) ldapEntryList {

	lineScanner := bufio.NewScanner(strings.NewReader(ldif))

	ldapEntries := make(ldapEntryList, 0)

	var currentLdapEntry *ldapEntry

	// Iterate through entries
	for lineScanner.Scan() {

		line := strings.TrimSpace(lineScanner.Text())

		// Ignore comments
		if len(line) > 0 && line[0] == '#' {
			continue
		}

		// Blank line. May mark the end of the entry. If so, add to entries and mark current entry as nil
		if len(line) == 0 {
			if currentLdapEntry != nil {
				// Add entry only if dn is not empty. Otherwise ignore
				if currentLdapEntry.dn != "" {
					// Add to the list of entries
					ldapEntries = append(ldapEntries, *currentLdapEntry)
				}
				currentLdapEntry = nil
			}
			continue
		}

		// Regular line. Append to current
		// Create entry if it was not existing yet
		if currentLdapEntry == nil {
			currentLdapEntry = &ldapEntry{
				attributes: make(map[string][]string),
			}
		}

		// A line MUST be of the form attrName: AttrValue
		attrAndValue := strings.Split(line, ":")
		if len(attrAndValue) != 2 {
			fmt.Println("[ERROR] Line not valid: ", line)
			os.Exit(1)
		}
		attr := strings.TrimSpace(attrAndValue[0])
		value := strings.TrimSpace(attrAndValue[1])

		if attr == "dn" {
			// Add the dn
			if currentLdapEntry.dn != "" {
				fmt.Println("[ERROR] Entry with two dn ", line)
				os.Exit(1)
			} else {
				currentLdapEntry.dn = value
			}
		} else {
			// Add an attribute
			// If non existing, crete a slice with a single value
			// If existing, append to the slice
			currentValue := currentLdapEntry.attributes[attr]
			if currentValue == nil {
				currentLdapEntry.attributes[attr] = []string{value}
			} else {
				currentLdapEntry.attributes[attr] = append(currentLdapEntry.attributes[attr], value)
			}
		}
	}

	// Treat the possibly last entry
	if currentLdapEntry != nil {
		// Add entry only if dn is not empty. Otherwise error
		if currentLdapEntry.dn != "" {
			ldapEntries = append(ldapEntries, *currentLdapEntry)
		} else {
			fmt.Println("[ERROR] Last entry is lackng dn ", currentLdapEntry)
			os.Exit(1)
		}
	}

	// Sort the entries
	sort.Sort(ldapEntries)

	return ldapEntries
}

// Generates an ldapmodify spec to make changes from currentLdif to targetLdif
func compareLdif(targetLdif ldapEntryList, currentLdif ldapEntryList) string {
	currentPos := 0
	targetPos := 0

	var builder strings.Builder

	// Target & Current are ldapentries(dn + attributes)
	// Target exists
	//	Current exists
	//	  compare entries
	//  Current does not exist
	//	  add
	// Target does not exist
	//	Current exists
	//	  delete
	//  Current does not exist
	//	  break

	var operation string
	for {
		if targetPos < len(targetLdif) {
			if currentPos < len(currentLdif) {
				if currentLdif[currentPos].dn < targetLdif[targetPos].dn {
					operation = "delete"
				} else if currentLdif[currentPos].dn > targetLdif[targetPos].dn {
					operation = "add"
				} else {
					operation = "compare"
				}
			} else {
				operation = "add"
			}
		} else {
			if currentPos < len(currentLdif) {
				operation = "delete"
			} else {
				break
			}
		}

		switch operation {
		case "add":
			builder.WriteString(targetLdif[targetPos].serializeDN())
			builder.WriteString("changetype: add\n")
			builder.WriteString(targetLdif[targetPos].serializeAttributes())
			builder.WriteString("\n")
			targetPos++
		case "compare":
			builder.WriteString(compareEntry(targetLdif[targetPos], currentLdif[currentPos]))
			currentPos++
			targetPos++
		case "delete":
			builder.WriteString(currentLdif[currentPos].serializeDN())
			builder.WriteString("changetype: delete\n")
			builder.WriteString("\n")
			currentPos++
		}
	}

	return builder.String()
}

// Generates an ldapmodify spec to make changes from the currentEntry attributes to the targetEntry attributes
func compareEntry(targetEntry ldapEntry, currentEntry ldapEntry) string {
	currentPos := 0
	targetPos := 0
	isEmpty := true

	var targetAttributes = targetEntry.getSortedAttributes()
	var currentAttributes = currentEntry.getSortedAttributes()

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("dn: %s\n", targetEntry.dn))
	builder.WriteString("changetype: modify\n")

	var operation string
	for {
		if targetPos < len(targetAttributes) {
			if currentPos < len(currentAttributes) {
				if currentAttributes[currentPos] < targetAttributes[targetPos] {
					operation = "delete"
				} else if currentAttributes[currentPos] > targetAttributes[targetPos] {
					operation = "add"
				} else {
					operation = "continue"
				}
			} else {
				operation = "add"
			}
		} else {
			if currentPos < len(currentAttributes) {
				operation = "delete"
			} else {
				break
			}
		}

		switch operation {
		case "add":
			builder.WriteString("-\n")
			builder.WriteString(fmt.Sprintf("add: %s\n", getAttributeName(targetAttributes[targetPos])))
			builder.WriteString(targetAttributes[targetPos])
			builder.WriteString("\n")
			targetPos++
			isEmpty = false
		case "continue":
			currentPos++
			targetPos++
		case "delete":
			builder.WriteString("-\n")
			builder.WriteString(fmt.Sprintf("delete: %s\n", getAttributeName(currentAttributes[currentPos])))
			builder.WriteString(currentAttributes[currentPos])
			builder.WriteString("\n")
			currentPos++
			isEmpty = false
		}
	}

	if isEmpty {
		return ""
	} else {
		builder.WriteString("\n")
		return builder.String()
	}
}

func getAttributeName(attrString string) string {
	kv := strings.Split(attrString, ":")
	return strings.TrimSpace(kv[0])
}
