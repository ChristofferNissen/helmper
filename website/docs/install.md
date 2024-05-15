---
sidebar_label: 'Install'
sidebar_position: 3
---

# Install

Simply pick the binary for your platform from the Release section on [GitHub](https://github.com/ChristofferNissen/helmper/releases/latest).

### Linux

```shell title="bash"
VERSION=$(curl -Lso /dev/null -w %{url_effective} https://github.com/christoffernissen/helmper/releases/latest | grep -o '[^/]*$')
curl -LO https://github.com/christoffernissen/helmper/releases/download/$VERSION/helmper-linux-amd64
chmod +x helmper-linux-amd64
sudo mv helmper-linux-amd64 /usr/local/bin/helmper
```

### Mac OS

```shell title="bash"
VERSION=$(curl -Lso /dev/null -w %{url_effective} https://github.com/christoffernissen/helmper/releases/latest | grep -o '[^/]*$')
curl -LO https://github.com/christoffernissen/helmper/releases/download/$VERSION/helmper-darwin-amd64
chmod +x helmper-darwin-amd64
sudo mv helmper-darwin-amd64 /usr/local/bin/helmper
```

### Windows

Extract the tar and launch the exe file.
