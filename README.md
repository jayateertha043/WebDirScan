# WebDirScan


WebDirScan is a tool for brute-forcing URIs (directories and files) on web servers by taking input directory to scan for files & directories recursively. It's written in Go and it's capable of multithreaded scanning. 


## Use Case
1. For Developers - Suppose, you are having a web server & want to know which of the files & directories are accessible to public.
2. For Security Researchers & BugBounty hunters - When testing for On-Premise products, you can hunt for sensitive directories & files for profit 🤑 !!!


## Installation
```
go install github.com/jayateertha043/WebDirScan@latest
```

## Installation from Source
1. Install Golang: https://golang.org/doc/install
2. Clone this repository: git clone https://github.com/jayateertha043/WebDirScan.git
3. Change to the directory where you cloned the repository: cd WebDirScan
4. Build the executable: go build
5. You can now run the executable: ./WebDirScan


## Usage
```
Usage of WebDirScan:
  -dir string
        Input Directory (default ".")
  -domain string
        Enter domain to scan (default "localhost")
  -headers string
        To use Custom Headers headers.json file
  -http string
        Enter HTTP ports (comma-separated)
  -https string
        Enter HTTPS ports (comma-separated)
  -out string
        Output Directory (default ".")
  -threads int
        Number of Threads (default 100)
  -timeout int
        Timeout for Request in Seconds (default 10)
  -verbose
        Verbose Output
```

## Author

👤 **Jayateertha G**

* Twitter: [@jayateerthaG](https://twitter.com/jayateerthaG)
* Github: [@jayateertha043](https://github.com/jayateertha043)

## License
WebDirScan is licensed under the MIT License. See LICENSE for more information.
