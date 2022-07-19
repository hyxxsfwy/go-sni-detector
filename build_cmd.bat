:: Windows 主机编译 Windows 客户端
SET CGO_ENABLED=0
SET GOOS=windows
SET GOARCH=amd64
go build -o .\bin\go-sni-detector_windows_amd64.exe

::Windows 主机编译 LINUX 客户端
SET CGO_ENABLED=0
SET GOOS=linux
SET GOARCH=amd64
go build -o .\bin\go-sni-detector_linux_amd64.bin
