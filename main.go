package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// go run main.go run <cmd> <args>
func main() {
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("help")
	}
}

func run() {
	fmt.Printf("Running %v \n", os.Args[2:])

	/**
	New containerized process can not be started without initializing new namespace.
	/proc/self/exe will call itself inside the new namespace
	*/
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	/**
	CLONE_NEWUTS: Unix Time Sharing System, isolate hostname and domain name
	CLONE_NEWPID: Process Ids, Isolate the PID number space
	CLONE_NEWNS: Creates a new mount system
	Unshareflags : Dont share mount space with the host
	*/
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	must(cmd.Run())
}

func child() {
	fmt.Printf("Running %v \n", os.Args[2:])

	cg()

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	must(syscall.Sethostname([]byte("container")))
	/**
	Changing root so that conatiner can have its own file system.
	*/
	must(syscall.Chroot("/home/rajurastogi/vagrant/ubunut-fs/ubuntu-fs"))
	must(os.Chdir("/"))                                // Changing directory to root
	must(syscall.Mount("proc", "proc", "proc", 0, "")) ///// Mounting the proc which is read by ps for all process related information

	must(cmd.Run())
	syscall.Unmount("/proc", 0)
}

/**
It creates a control group in main host restricting the number of process for this container.
*/
func cg() {
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")
	os.Mkdir(filepath.Join(pids, "test"), 0755)
	must(ioutil.WriteFile(filepath.Join(pids, "test/pids.max"), []byte("20"), 0700)) /// 20 max process
	// Removes the new cgroup in place after the container exits
	must(ioutil.WriteFile(filepath.Join(pids, "test/notify_on_release"), []byte("1"), 0700))
	//// every process is written in this file which helps in keeping track of number of process on hosts
	must(ioutil.WriteFile(filepath.Join(pids, "test/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
