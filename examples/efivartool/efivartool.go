package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.pdmccormick.com/linuxuapi/fs/efivarfs"
)

func findName(names efivarfs.Names, filename, guid string) efivarfs.Name {
	name, ok := efivarfs.NameFromFilename(filename)
	if !ok {
		name = efivarfs.Name{
			GUID: guid,
			Id:   filename,
		}
	}

	var matches efivarfs.Names

	name.GUID = strings.ToLower(name.GUID)
	name.Id = strings.ToLower(name.Id)

	for _, n := range names {
		var (
			idMatch   = name.Id == strings.ToLower(n.Id)
			guid      = strings.ToLower(n.GUID)
			guidMatch = name.GUID == guid || strings.HasPrefix(guid, name.GUID)
		)

		if idMatch && (guidMatch || name.GUID == "") {
			matches.Append(n)
		}
	}

	if len(matches) == 0 {
		log.Fatalf("no variables match name `%s`", filename)
	}

	if len(matches) > 1 {
		log.Printf("multiple GUID's match for name `%s`:\n\n", filename)
		matches.SortByGUID()

		for _, m := range matches {
			fmt.Printf("%s\n", m)
		}

		os.Exit(1)
	}

	return matches[0]
}

func main() {
	var (
		mountFlag    = flag.String("mount", efivarfs.DefaultMountPath, "use `path` as efivarfs mount point")
		allFlag      = flag.Bool("all", false, "get all variables")
		globFlag     = flag.String("glob", "", "glob variable name")
		getFlag      = flag.String("get", "", "get variable by file`name` (id-guid), or just the id portion plus the -guid flag")
		setFlag      = flag.String("set", "", "set variable by file`name` (id-guid), or just the id portion plus the -guid flag")
		removeFlag   = flag.String("rm", "", "remove variable by file`name` (id-guid)")
		createFlag   = flag.Bool("create", false, "create variable if it does not exist")
		guidFlag     = flag.String("guid", "", "specify the GUID portion of the name")
		writeFlag    = flag.String("w", "", "write value to `name`")
		noDumpFlag   = flag.Bool("nodump", false, "do not dump contents of variables")
		sortIdFlag   = flag.Bool("sortid", false, "sort names by Id before GUID")
		showPathFlag = flag.Bool("showpath", false, "show full path to efivarfs file")
		dataFlag     = flag.String("data", "", "read data to set from file`name`")
	)

	flag.Parse()

	var varfs = efivarfs.FSFromMount(*mountFlag)

	names, err := varfs.GlobNames(*globFlag, *guidFlag)
	if err != nil {
		log.Fatalf("GlobNames: %s", err)
	}

	if id := *removeFlag; id != "" {
		name := findName(names, id, *guidFlag)

		log.Printf("removing %s (%s)", name, filepath.Join(*mountFlag, name.Filename()))

		if err := varfs.RemoveName(name); err != nil {
			log.Fatalf("RemoveName: %s", err)
		}

		return
	}

	if id := *setFlag; id != "" {
		var (
			name efivarfs.Name
			val  efivarfs.Value
		)

		if *createFlag {
			name0, ok := efivarfs.NameFromFilename(id)
			if !ok {
				log.Fatalf("malformed name: %s", id)
			}

			name = name0
			val.Attrs = efivarfs.NonVolatile | efivarfs.BootServiceAccess | efivarfs.RuntimeAccess
		} else {
			name = findName(names, id, *guidFlag)

			val0, err := varfs.GetValue(name)
			if err != nil {
				log.Fatalf("%s", err)
			}

			val = *val0
		}

		if *dataFlag == "" {
			log.Fatalf("missing `-data`")
		}

		data, err := os.ReadFile(*dataFlag)
		if err != nil {
			log.Fatalf("ReadFile: %s", err)
		}

		val.Data = data

		if err := varfs.SetValue(name, &val); err != nil {
			log.Fatalf("SetValue %s: %s", name, err)
		}

		return
	}

	if id := *getFlag; id != "" {
		name := findName(names, id, *guidFlag)

		const _ = `
		if guid := *guidFlag; guid != "" {
			name = efivarfs.Name{
				GUID: guid,
				Id:   id,
			}
		} else {
			var ok bool
			name, ok = efivarfs.NameFromFilename(id)
			if !ok {
				names, err := varfs.ListNames()
				if err != nil {
					log.Fatalf("ListNames: %s", err)
				}

				var matches []efivarfs.Name

				for _, name := range names {
					if strings.ToLower(name.Id) == strings.ToLower(id) {
						matches = append(matches, name)
					}
				}

				if len(matches) == 0 {
					log.Fatalf("no variables match name %s", id)
				}

				if len(matches) > 1 {
					log.Printf("multiple GUID's match for name %s:\n\n", id)
					for _, m := range matches {
						fmt.Printf("%s\n", m)
					}

					return
				}

				name = matches[0]
			}
		}
			`

		val, err := varfs.GetValue(name)
		if err != nil {
			log.Fatalf("%s", err)
		}

		var pathSuffix string
		if *showPathFlag {
			pathSuffix = " " + filepath.Join(*mountFlag, name.Filename())
		}

		fmt.Printf("%s [%s] (%d bytes)%s\n", name, val.Attrs, len(val.Data), pathSuffix)
		if out := *writeFlag; out != "" {
			if err := os.WriteFile(out, val.Data, 0o660); err != nil {
				log.Fatalf("WriteFile %s: %s", out, err)
			} else {
				fmt.Printf("wrote %s\n", out)
			}
		} else if !*noDumpFlag {
			fmt.Println(hex.Dump(val.Data))
		}

		return
	}

	if *allFlag {
		vars, err := varfs.GetAll()
		if err != nil {
			log.Printf("GetAll: %s", err)
		}

		for _, v := range vars {
			var (
				name  = v.Name
				attrs = fmt.Sprintf("%#02x:[%s]", uint32(v.Attrs), v.Attrs)
			)
			fmt.Printf("%s %-20s %s (%d bytes)\n", name.GUID, attrs, name.Filename(), len(v.Data))
			if !*noDumpFlag {
				fmt.Println(hex.Dump(v.Data))
			}
		}

		return
	}

	// List
	if *sortIdFlag {
		names.SortById()
	}

	for _, name := range names {
		fmt.Printf("%s\n", name)
	}
}
