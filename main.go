package main
// go run main.go run <cmd> <params>
import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

//cgroups - limit what resources the container can use

func main() {

	switch os.Args[1] {

	case "run":
		run()
	case "child":
		child()
	default:
		panic("bad command")
	}
}

func run() {

	fmt.Printf("Command: %v as %d\n", os.Args[2:], os.Getpid())

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	//creating namespaces -
	// CLONE_NEWUTS (Unix Time Sharing System - give new hostname)
	// CLONE_NEWPID (give pid starting from 1)
	// CLONE_NEWNS (mount)
	// Unshareflags: tell kernel that don't share the mount created in the container with host
	// 	  check - mount | grep proc (host)
	// 			  don't view the proc shared in the container
	//			  don't clutter up the mount command in host

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	cmd.Run()

}

func child() {

	fmt.Printf("Command: %v as %d\n", os.Args[2:], os.Getpid())

	cg()

	syscall.Sethostname([]byte("container"))
	syscall.Chroot("/home/ubuntu-fs")
	//change the root of container to ubuntu-fs
	//to check root
	//sleep 100 - inside container
	// ps -C sleep
	//ls -l /proc/pid/root - host

	syscall.Chdir("/")

	//tell the kernel to mount /proc as ps for this container
	//do ps and check
	syscall.Mount("proc", "proc", "proc", 0, "")

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()

	syscall.Unmount("/proc", 0)

}

func cg() {

	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")
	err := os.Mkdir(filepath.Join(pids, "ash"), 0755)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	memory := filepath.Join(cgroups,"memory")
	err = os.Mkdir(filepath.Join(memory,"ash"),0755)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	cpu := filepath.Join(cgroups,"cpu")
	err = os.Mkdir(filepath.Join(cpu,"ash"),0755)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	
	//setting max number of processes to 20
	must(ioutil.WriteFile(filepath.Join(pids, "ash/pids.max"), []byte("20"), 0700))
	//setting memory limit as 500 mb
	must(ioutil.WriteFile(filepath.Join(memory, "ash/memory.limit_in_bytes"),[]byte("524288000"),0700))
	//setting memory swap limit as 500 mb
	must(ioutil.WriteFile(filepath.Join(memory, "ash/memory.memsw.limit_in_bytes"),[]byte("524288000"),0700))
	//setting cpu.cfs_period_us to 1 s (specifies a period of time in microseconds (µs, represented here as "us") for how regularly a cgroup's access to CPU resources should be reallocated. 
	//If tasks in a cgroup should be able to access a single CPU for 0.2 seconds out of every 1 second, set cpu.cfs_quota_us to 200000 and cpu.cfs_period_us to 1000000)
	must(ioutil.WriteFile(filepath.Join(cpu, "ash/cpu.cfs_period_us"),[]byte("1000000"),0700))
	must(ioutil.WriteFile(filepath.Join(cpu, "ash/cpu.cfs_quota_us"),[]byte("2000000"),0700))

	must(ioutil.WriteFile(filepath.Join(pids, "ash/notify_on_release"), []byte("1"), 0700))
	must(ioutil.WriteFile(filepath.Join(memory, "ash/notify_on_release"), []byte("1"), 0700))
	must(ioutil.WriteFile(filepath.Join(cpu, "ash/notify_on_release"), []byte("1"), 0700))
	
	//tell kernel that this process is aslo with the same cgroups
	must(ioutil.WriteFile(filepath.Join(pids, "ash/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
	must(ioutil.WriteFile(filepath.Join(memory, "ash/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
	must(ioutil.WriteFile(filepath.Join(cpu, "ash/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}

//to check cg
// /sys/fs/cgroup/ash (in host)
// - cat pids.max (expect 20)
// inside container - able to create only 20 processes
// :() { : | : & }; : (fork bomb)
// cat pids.cuurent (expect 20)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
