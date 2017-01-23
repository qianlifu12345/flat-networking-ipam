package store

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/qianlifu12345/flat-networking-ipam/rest-service/model"

	cmap "github.com/orcaman/concurrent-map"

	"regexp"

	"net"

	"github.com/astaxie/beego"
)

var defaultDataDir = "./"

func init() {
	dataDir := beego.AppConfig.String("dataPath")
	if dataDir != "" {
		defaultDataDir = dataDir
	}
}

// Netstore st
type Netstore struct {
	dataDir string
}

// NewIPNet create a new network file for specified network
func NewIPNet(ip string, network string) error {
	if err := os.MkdirAll(defaultDataDir, 0644); err != nil {
		return err
	}
	fname := filepath.Join(defaultDataDir, strings.Replace(ip, "/", "#", -1))
	f, err := os.OpenFile(fname, os.O_RDWR|os.O_EXCL|os.O_CREATE, 0644)
	defer f.Close()
	if os.IsExist(err) {
		return errors.New("save error")
	}
	if err != nil {
		return err
	}
	if _, err := f.WriteString(network); err != nil {
		return err
	}
	return nil
}

// ReserveIP save used ip into local file
func ReserveIP(subnet, ip string) error {
	fname := filepath.Join(defaultDataDir, strings.Replace(subnet, "/", "#", -1))
	f, err := os.OpenFile(fname, os.O_RDWR|os.O_APPEND, 0644)
	defer f.Close()
	if err != nil {
		return err
	}
	if _, err := f.WriteString("\n" + ip); err != nil {
		return err
	}
	return nil
}

// Store save entire subnetwork into local file
func Store(subnet *model.Subnetwork) error {
	fname := filepath.Join(defaultDataDir, strings.Replace(subnet.Subnet.String(), "/", "#", -1))
	fnameTmp := fname + ".tmp"
	f, err := os.OpenFile(fnameTmp, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0777)
	defer f.Close()
	if err != nil {
		return err
	}
	if _, err := f.WriteString(subnet.String()); err != nil {
		return err
	}
	for _, ip := range subnet.Ips {
		if _, err := f.WriteString("\n" + ip.String()); err != nil {
			return err
		}
	}
	err = f.Sync()
	if err != nil {
		return err
	}
	os.Remove(fname)
	f.Close()
	err = os.Rename(fnameTmp, fname)
	if err != nil {
		return err
	}
	return nil
}

var subnetRex, _ = regexp.Compile("([a-z]+)=([a-z]+)")
var ipv4Rex, err = regexp.Compile("^((0|1[0-9]{0,2}|2[0-9]{0,1}|2[0-4][0-9]|25[0-5]|[3-9][0-9]{0,1})\\.){3}(0|1[0-9]{0,2}|2[0-9]{0,1}|2[0-4][0-9]|25[0-5]|[3-9][0-9]{0,1})((\\/)?\\/([0-9]|[1-2][0-9]|3[0-2])|)$")

// LoadIPAMConfig load IPAM configurations from local store
func LoadIPAMConfig() cmap.ConcurrentMap {
	subnetMap := cmap.New()
	filepath.Walk(defaultDataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		defer f.Close()
		if err != nil {
			return err
		}

		input := bufio.NewScanner(f)
		subnetwork := model.Subnetwork{}
		subnetwork.Ips = []net.IP{}

	ll:
		for input.Scan() {
			line := strings.TrimSpace(input.Text())
			if line == "" {
				continue
			}

			switch substrs := strings.SplitN(line, "=", 2); substrs[0] {
			case "subnet":
				subnet, err := model.ParseCIDR(substrs[1])
				if err != nil {
					return nil
				}
				subnetwork.Subnet = (*model.IPNet)(subnet)
				continue ll
			case "gateway":
				subnetwork.Gateway = net.ParseIP(substrs[1])
				continue ll
			case "rangeStart":
				subnetwork.RangeStart = net.ParseIP(substrs[1])
				continue ll
			case "rangeEnd":
				subnetwork.RangeEnd = net.ParseIP(substrs[1])
				continue ll
			}
			if ipv4Rex.MatchString(line) {
				subnetwork.Ips = append(subnetwork.Ips, net.ParseIP(line))
			}

		}
		if subnetwork.Subnet != nil {
			subnetMap.Set(subnetwork.Subnet.String(), &subnetwork)
		}

		return nil
	})
	return subnetMap
}
