package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"gitlab.cloud.vtblife.ru/vtblife/mobile/common/gripmock/stub"
)

func main() {
	outputPointer := flag.String("o", "", "directory to output server.go. Default is $GOPATH/src/")
	grpcPort := flag.String("grpc-port", "4770", "Port of gRPC tcp server")
	grpcBindAddr := flag.String("grpc-listen", "", "Adress the gRPC server will bind to. Default to localhost, set to 0.0.0.0 to use from another machine")
	adminport := flag.String("admin-port", "4771", "Port of stub admin server")
	adminBindAddr := flag.String("admin-listen", "", "Adress the admin server will bind to. Default to localhost, set to 0.0.0.0 to use from another machine")
	stubPath := flag.String("stub", "", "Path where the stub files are (Optional)")
	imports := flag.String("imports", "/protobuf", "comma separated imports path. default path /protobuf is where gripmock Dockerfile install WKT protos")
	// for backwards compatibility
	if os.Args[1] == "gripmock" {
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}

	flag.Parse()
	fmt.Println("Starting GripMock")

	output := *outputPointer
	if output == "" {
		if os.Getenv("GOPATH") == "" {
			log.Fatal("output is not provided and GOPATH is empty")
		}
		output = os.Getenv("GOPATH") + "/src"
	}

	// for safety
	output += "/"
	if _, err := os.Stat(output); os.IsNotExist(err) {
		os.Mkdir(output, os.ModePerm)
	}

	// run admin stub server
	stub.RunStubServer(stub.Options{
		StubPath: *stubPath,
		Port:     *adminport,
		BindAddr: *adminBindAddr,
	})

	// parse proto files
	protoPaths := flag.Args()

	if len(protoPaths) == 0 {
		log.Fatal("Need atleast one proto file")
	}

	importDirs := strings.Split(*imports, ",")

	// generate pb.go and grpc server based on proto
	paths := generateProtoc(output, protocParam{
		protoPath:   protoPaths,
		adminPort:   *adminport,
		grpcAddress: *grpcBindAddr,
		grpcPort:    *grpcPort,
		output:      output,
		imports:     importDirs,
	})

	// build the server
	buildServer(output, paths)

	// and run
	run, runerr := runGrpcServer(output)

	var term = make(chan os.Signal)
	signal.Notify(term, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGINT)
	select {
	case err := <-runerr:
		log.Fatal(err)
	case <-term:
		fmt.Println("Stopping gRPC Server")
		run.Process.Kill()
	}
}

func getProtoName(path string) string {
	paths := strings.Split(path, "/")
	filename := paths[len(paths)-1]
	return strings.Split(filename, ".")[0]
}

type protocParam struct {
	protoPath   []string
	adminPort   string
	grpcAddress string
	grpcPort    string
	output      string
	imports     []string
}

func generateProtoc(output string, param protocParam) []string {
	protodirs := strings.Split(param.protoPath[0], "/")
	protodir := ""
	if len(protodirs) > 0 {
		protodir = strings.Join(protodirs[:len(protodirs)-1], "/") + "/"
	}

	args := []string{"-I", protodir}
	// include well-known-types
	for _, i := range param.imports {
		args = append(args, "-I", i)
	}
	args = append(args, param.protoPath...)
	args = append(args, "--go_out=plugins=grpc:"+param.output)
	args = append(args, fmt.Sprintf("--gripmock_out=admin-port=%s,grpc-address=%s,grpc-port=%s:%s",
		param.adminPort, param.grpcAddress, param.grpcPort, param.output))
	protoc := exec.Command("protoc", args...)
	protoc.Stdout = os.Stdout
	protoc.Stderr = os.Stderr
	err := protoc.Run()
	if err != nil {
		log.Fatal("Fail on protoc ", err)
	}

	paths := []string{"/go/src/auth_flow_service.pb.go"}

	// change package to "main" on generated code
	for _, proto := range param.protoPath {
		
		file := strings.Split(proto, "/")
		newFile := make([]string, len(file) + 1)
		condition := false
		for i, s := range file {
			if i == 1 {
			  newFile[i] = "go"
			  continue
			}
			if i == 0 {
			  continue
			}
			if i == 2 {
			  newFile[i] = "src"
			  continue
			}
			if s == "vtblife" {
				condition = true
			}
			if condition {
				newFile[i+1] = s
			}
		}

		newFile[len(file)] = getProtoName(newFile[len(file)]) + ".pb.go"
		arrayWithoutEmpty := delete_empty(newFile)
		newPath := strings.Join(arrayWithoutEmpty[:], "/")
		log.Printf(newPath)
		paths = append(paths, newPath)
		// lsCom := exec.Command("ls")
		// lsCom.Stdout = os.Stdout
		// lsCom.Stderr = os.Stderr
		// lsCom.Run()
	// 	sed := exec.Command("sed", "-i", `s/^package \w*$/package main/`, param.output+protoname+".pb.go")
	// 	sed.Stderr = os.Stderr
	// 	sed.Stdout = os.Stdout
	// 	err = sed.Run()
	// 	if err != nil {
	// 		log.Fatal("Fail on sed")
	// 	}
	}
	copyCom := exec.Command("find . -name *.go")
	copyCom.Stdout = os.Stdout
	copyCom.Stderr = os.Stderr
	copyCom.Run()
	log.Printf(copyCom.Stdout)
	pathsWithoutFirst := delete_fisrt(paths)
	return pathsWithoutFirst
}

func delete_empty (s []string) []string {
    var r []string
    for _, str := range s {
        if str != "" {
            r = append(r, str)
        }
    }
    return r
}

func delete_fisrt (s []string) []string {
    var r []string
    for index, str := range s {
        if index != 1 {
            r = append(r, str)
        }
    }
    return r
}

func buildServer(output string, protoPaths []string) {
	args := []string{"build", "-o", output + "grpcserver", output + "server.go"}
	// for _, path := range protoPaths {
	// 	args = append(args, path)
	// }
	args = append(args, "go/src/auth/v2/model/auth_flow.pb.go")
	build := exec.Command("go", args...)
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	err := build.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func runGrpcServer(output string) (*exec.Cmd, <-chan error) {
	run := exec.Command(output + "grpcserver")
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	err := run.Start()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("grpc server pid: %d\n", run.Process.Pid)
	runerr := make(chan error)
	go func() {
		runerr <- run.Wait()
	}()
	return run, runerr
}
