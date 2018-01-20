package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

func main() {
	fmt.Println("Searching Atom versions...")
	page, err := getLatestReleasePage()
	if err != nil {
		checkError(err, err.Error())
	}

	localVersion, err := getLocalVersion() // getLocalVersion()
	if err != nil {
		checkError(err, err.Error())
	} else {
		fmt.Println("Found local version: " + localVersion)
	}

	latestVersion, downloadLink, err := parsePage(page)
	if err != nil {
		checkError(err, err.Error())
	}

	if localVersion == latestVersion {
		fmt.Println("You're already up-to-date !")
	} else {
		fmt.Println("Found newer version: " + latestVersion)
		fmt.Println("Downloading latest version...")

		done := make(chan bool)
		go func() {
			downloadFile(downloadLink)
			done <- true
		}()
		statusBar(done)

		fmt.Println("\nDownload successfully completed")
		fmt.Println("Unpacking atom...")
		unpackFile()
	}
}

// getLocalVersion uses `atom --version` command to get installed version
func getLocalVersion() (string, error) {
	cmd := exec.Command("atom", "--version")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return "", errors.New("Atom Editor is not installed")
	}
	result := strings.SplitN(out.String(), "\n", 2)[0]
	result = strings.SplitN(result, ":", 2)[1]

	return strings.TrimSpace(result), nil
}

// parsePage parses the page from `https://github.com/atom/atom/releases/latest`
// and retrieves the version number and the download link
func parsePage(page io.Reader) (string, string, error) {
	var version string
	var link string

	doc, err := html.Parse(page)
	if err != nil {
		return "", "", fmt.Errorf("Error while Parsing File: %v", err.Error())
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h1" {
			for _, attr := range n.Attr {
				match, _ := regexp.MatchString("release-title", attr.Val)
				if attr.Key == "class" && match {
					version = n.FirstChild.NextSibling.FirstChild.Data
					break
				}
			}
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				match, _ := regexp.MatchString(`.*/atom-amd64\.deb`, attr.Val)
				if attr.Key == "href" && match {
					link = "https://github.com" + attr.Val
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if version != "" && link != "" {
				break
			}
			f(c)
		}
	}

	f(doc)
	if version == "" || link == "" {
		return "", "", errors.New("HTML has changed, update your code")
	}

	return version, link, nil
}

// getLatestReleasePage gets the html file provided
// from `https://github.com/atom/atom/releases/latest` page
func getLatestReleasePage() (io.Reader, error) {
	resp, err := http.Get("https://github.com/atom/atom/releases/latest")
	if err != nil {
		return nil, errors.New("Can't get connection")
	}

	return resp.Body, nil
}

// downloadFile prints the usual `atom-amd64.deb` into
// a file created in `/tmp/atom_latest.deb`
func downloadFile(downloadLink string) {
	output, err := os.Create("/tmp/atom_latest.deb")
	if err != nil {
		checkError(err, "Can't create file...")
	}
	defer output.Close()

	resp, err := http.Get(downloadLink)
	if err != nil {
		checkError(err, "Can't get file...")
	}
	defer resp.Body.Close()

	n, err := io.Copy(output, resp.Body)
	if err != nil {
		checkError(err, "Error while downloading...")
	}

	fmt.Printf("\n%d/%d bytes downloaded.\n", n, resp.ContentLength)
}

// unpackFile uses `sudo dpkg --install [file]` to unpack
func unpackFile() {
	cmd := exec.Command("/bin/bash", "-c", "sudo dpkg --install /tmp/atom_latest.deb")
	err := cmd.Run()
	if err != nil {
		checkError(err, "Can't unpack file...")
	}

	version, err := getLocalVersion()
	if err != nil {
		checkError(err, err.Error())
	}

	fmt.Printf("New version %v installed !\nHave a nice day ! ;-)\n", version)
}

func statusBar(done <-chan bool) {
	start := ">\r"
	waitLine := start
	var stop bool
	for {
		fmt.Print(waitLine)
		select {
		case <-done:
			stop = true
		default:
			time.Sleep(time.Second)
		}
		if stop {
			break
		}
		append := "="
		waitLine = fmt.Sprintf("%v%v", append, waitLine)
		if len(waitLine) == 20 {
			waitLine = start
		}
	}
	fmt.Println()
}

func checkError(err error, msg interface{}) {
	fmt.Println(msg)
	os.Exit(1)
}
