package lib

import (
	"bufio"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var processNames = []string{
	"StarCraft.exe",                          // Windows (maybe? untested.)
	"StarCraft.app/Contents/MacOS/StarCraft", // macOS
}

// getProcessID runs 'ps aux' or 'tasklist' and filters the output to find the StarCraft process.
// If found, it returns only the pid as a number, or -1 otherwise.
func getProcessID(procMatches []string) (int, error) {
	procMatchesLc := make([]string, len(procMatches))
	for i, match := range procMatches {
		procMatchesLc[i] = strings.ToLower(match)
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("tasklist")
	} else {
		cmd = exec.Command("ps", "aux")
	}

	out, err := cmd.Output()
	if err != nil {
		return -1, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		for _, proc := range procMatchesLc {
			if strings.Contains(strings.ToLower(line), proc) {
				fields := strings.Fields(line)
				return strconv.Atoi(fields[1])
			}
		}
	}

	return -1, nil
}

// getOpenPorts returns an array of open ports for a given pid.
func getOpenPorts(pid int) ([]int, error) {
	var cmd *exec.Cmd
	portsSet := make(map[int]struct{})

	if runtime.GOOS == "windows" {
		cmd = exec.Command("netstat", "-on")
		out, _ := cmd.Output()
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, strconv.Itoa(pid)) {
				fields := strings.Fields(line)
				if len(fields) < 5 {
					continue
				}
				nameField := fields[1]
				if !strings.Contains(nameField, "localhost") && !strings.Contains(nameField, "127.0.0.1") {
					continue
				}

				nameFieldParts := strings.Split(nameField, ":")
				if len(nameFieldParts) < 2 {
					continue
				}
				portStr := nameFieldParts[len(nameFieldParts)-1]
				port, err := strconv.Atoi(portStr)
				if err != nil {
					continue
				}

				portsSet[port] = struct{}{}
			}
		}
	} else {
		cmd = exec.Command("lsof", "-aPi", "-p", strconv.Itoa(pid))

		out, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}

			var nameField string
			var pidField string
			if runtime.GOOS == "windows" {
				nameField = fields[0]
				pidField = fields[1]
			} else {
				nameField = fields[8]
				pidField = fields[1]
			}

			if !strings.Contains(nameField, "localhost") && !strings.Contains(nameField, "127.0.0.1") {
				continue
			}

			nameFieldParts := strings.Split(nameField, ":")
			if len(nameFieldParts) < 2 {
				continue
			}

			portStr := nameFieldParts[len(nameFieldParts)-1]
			port, err := strconv.Atoi(portStr)
			if err != nil {
				continue
			}

			pidInt, err := strconv.Atoi(pidField)
			if err != nil || pidInt != pid {
				continue
			}

			portsSet[port] = struct{}{}
		}
	}

	var ports []int
	for port := range portsSet {
		ports = append(ports, port)
	}

	return ports, nil
}

// findWorkingPort finds which one of a list of ports is the correct one by querying them all.
func findWorkingPort(ports []int) (int, error) {
	var wg sync.WaitGroup
	portCh := make(chan int)

	for _, port := range ports {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()

			url := fmt.Sprintf("http://127.0.0.1:%d/web-api/v1/leaderboard/12931?offset=0&length=100", port)
			resp, err := http.Get(url)
			if err != nil || resp.StatusCode != http.StatusOK {
				return
			}
			portCh <- port
		}(port)
	}

	go func() {
		wg.Wait()
		close(portCh)
	}()

	for port := range portCh {
		return port, nil
	}

	return -1, fmt.Errorf("no working port found")
}

// getProcessInfo finds the StarCraft process ID and then finds the open port we need to use.
func GetProcessInfo(onlyGetProcessID bool) (int, int, error) {
	proc, err := getProcessID(processNames)

	if err != nil {
		return -1, -1, err
	}

	if proc == -1 || onlyGetProcessID {
		return proc, -1, nil
	}

	// Find the port being used by the API.
	ports, err := getOpenPorts(proc)
	if err != nil {
		return -1, -1, err
	}

	port, err := findWorkingPort(ports)
	if err != nil {
		return -1, -1, err
	}

	return proc, port, nil
}
