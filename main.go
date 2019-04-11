package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

const(
	infoLevel = "info"
	accessId = ""
	accessSecret = ""
)

var(
	allSubAppList map[string]map[string]string = map[string]map[string]string{
		"food0001": map[string]string{
			// ios-android
			"prelaunch": "0-0", "sub": "1-1"},
		"upload-nexa-test":map[string]string{
			// ios-android
		"prelaunch": "0-0"}}

	mainappSubappResourceType map[string][]string = map[string][]string{
		"food0001": []string{
			".txt", ".zip", ".zip.txt", "lua.zip", "lua.zip.txt", "raw.zip",
			"raw.zip.txt"},
		"upload-nexa-test": []string{
			".txt", ".zip", ".zip.txt", "lua.zip", "lua.zip.txt", "raw.zip",
			"raw.zip.txt"},
	}
)

func downloadOnlySubapp(bucket, subapp, stage, platform string,
	currentDir string) []map[string]string{
	var(
		wg sync.WaitGroup
		version string
		resourceType string
		destPath string
		chMessage chan map[string]string
		subApp string
		cmdPath string
		configPath string
		result []map[string]string
	)

	configPath = path.Join(currentDir, "tools", ".ossutilconfig")
	cmdPath = path.Join(currentDir, "tools", "ossutilmac64")
	chMessage = make(chan map[string]string,
		len(mainappSubappResourceType[bucket]))
	// aliyun only support ios apps
	version = strings.Split(allSubAppList[bucket][subapp], "-")[0]
	for _, resourceType =
	range mainappSubappResourceType[bucket]{
		wg.Add(1)
		subApp = subapp
		if subapp == "prelaunch"{
			subApp = "mainapp"
		}
		destPath = "oss://" + bucket + "/" + platform + "/" + stage + "/" +
			version + "/" + subApp +resourceType
		go func(commandName string, params ...string){
			var (
				cmd *exec.Cmd
				stdout io.ReadCloser
				err error
				line string
			)

			cmd = exec.Command(commandName, params...)
			if stdout, err = cmd.StdoutPipe();err != nil{
				fmt.Println(err)
				return
			}

			cmd.Start()
			reader := bufio.NewReader(stdout)

			for {
				if line, err = reader.ReadString('\n');
					err != nil{
					if io.EOF == err{
						fmt.Println("Finished to read data")
						break
					}else{
						fmt.Println("Error happend:", err)
						return
					}
				}else{
					if strings.HasPrefix(line,
						"Content-Length"){
						chMessage <- map[string]string{
							destPath:strings.Trim(strings.Split(line,
								":")[1], "[ \n]")}
						break
					}
				}
			}
			cmd.Wait()
			wg.Done()
			return
		}(cmdPath, "stat", destPath,
			"--loglevel", infoLevel, "-c", configPath,
			"-k", accessSecret, "-i",  accessId)
	}

	wg.Wait()
	close(chMessage)
	for item := range chMessage{
		result = append(result, item)
	}
	return result
}


func main(){
	var (
		bucket *string
		subapp *string
		mode *string
		platform *string
		dataPath string
		stage *string
		result []map[string]string
		currentDir string
		err error
		_subapp string
		f *os.File
	)
	if currentDir, err = os.Getwd();err != nil{
		panic("Error happend when upload data: get current dir error")
	}
	bucket = flag.String("bucket", "", "the bucket you " +
		"want to get info from")
	subapp = flag.String("subapp", "", "the subapp you " +
		"want to get info")
	// if mode == "all", means get all subapp listed above info
	mode = flag.String("mode", "", "the mode you " +
		"want to get info")
	stage = flag.String("stage", "", "the stage you " +
		"want to get info")
	platform = flag.String("platform", "", "the platform " +
		"you want to get info")
	flag.Parse()

	if *platform != "ios"{
		fmt.Println("Current platform not support:", *platform)
		os.Exit(1)
	}

	if *stage != "test" && *stage != "production"{
		fmt.Println("Current stage not support:", *stage)
		os.Exit(1)
	}

	if *subapp != "" && *mode != "all"{
		if _, ok := allSubAppList[*bucket]; !ok {
			fmt.Println("Mainapp not exists or mainapp use different name" +
				" with bucket:", *subapp)
			os.Exit(1)
		}else{
			dataPath = path.Join(currentDir, "data",
				strings.Join([]string{*bucket, *subapp, *platform, *stage},
				"-")) + ".json"
			if _, ok := allSubAppList[*bucket][*subapp]; !ok {
				fmt.Println("Subapp not exists:", *subapp)
				os.Exit(1)
			}else{
				if _, ok := mainappSubappResourceType[*bucket]; !ok {
					fmt.Println("Get resource type failed, mainapp not " +
						"exists or mainapp use different name with bucket:",
						*subapp)
					os.Exit(1)
				}else{
					result = downloadOnlySubapp(*bucket, *subapp, *stage,
						*platform, currentDir)
					}
				}
			}

		}else if *subapp == "" && *mode == "all"{
			dataPath = path.Join(currentDir, "data",
				strings.Join([]string{*bucket, *mode, *platform, *stage},
					"-")) + ".json"
			for _subapp, _ = range allSubAppList[*bucket]{
				result = append(result, downloadOnlySubapp(*bucket, _subapp, 
          *stage, *platform, currentDir)...)
			}
		}else{
		fmt.Println("Not Support both subapp and mode=all")
		os.Exit(1)
		}
	if f, err = os.OpenFile(dataPath,
		os.O_WRONLY | os.O_CREATE,0777); err != nil{
		fmt.Println("打开文件错误：", dataPath)
	}else{
		enc := json.NewEncoder(f)
		enc.SetIndent("", "    ")
		if err := enc.Encode(&result); err != nil {
			log.Println(err)
		}
	}
}
