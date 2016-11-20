package model

import (
	"bytes"
	"fmt"
	"net"
	// "github.com/containernetworking/cni/pkg/types"
	// "github.com/containernetworking/cni/pkg/ip"
	"encoding/json"
	"math/big"
	"errors"
)

// Subnetwork define
type Subnetwork struct {
	Subnet         IPNet    `json:"subnet"`
	RangeStart     net.IP   `json:"range-start,omitempty"`
	RangeEnd       net.IP   `json:"range-end,omitempty"`
	Gateway        net.IP   `json:"gateway,omitempty"`
	DNS            net.IP   `json:"dns,omitempty"`
	Ips            []net.IP //`json:"-"`
	LastReservedIP net.IP   `json:"-"`
}

// String stringer for Subnetwork".
func (s *Subnetwork) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "subnet=%v\n", s.Subnet.String())
	fmt.Fprintf(&buf, "gateway=%v\n", s.Gateway)
	fmt.Fprintf(&buf, "dns=%v\n", s.DNS)
	fmt.Fprintf(&buf, "rangeStart=%v\n", s.RangeStart)
	fmt.Fprintf(&buf, "rangeEnd=%v\n", s.RangeEnd)
	return buf.String()
}

// NextIP find
func (s *Subnetwork) NextIP() (net.IP, error) {
	start, end, err := networkRange((*net.IPNet)(&s.Subnet))
	if err != nil {
		return nil, err
	}
	if s.RangeStart != nil {
		start = s.RangeStart
	}
	if s.RangeEnd != nil {
		end = s.RangeEnd
	}
	if s.LastReservedIP == nil {
		s.LastReservedIP = start
	}
	return findNextIPInRange(&start, &end, &s.LastReservedIP, &s.Ips)
}

func findNextIPInRange(start, end, curIP *net.IP, ips *[]net.IP) (net.IP,error) {
	newIP := next(start, end, curIP)
	for Contains(&newIP,ips) {
		newIP = next(start, end, &newIP)
		if newIP.Equal(*curIP){
			return nil,errors.New("no available IP could be apply")
		}
	}
	return newIP,nil
}

func next(start, end, curIP *net.IP) net.IP{
	if (*curIP).Equal(*end) {
		return *start
	}
	i := ipToInt(*curIP)
	newIP := intToIP(i.Add(i, big.NewInt(1)))
	return newIP
}

func Contains(curIP *net.IP, ips *[]net.IP) bool{
	for _, k := range *ips {
		if curIP.Equal(k) {
			return true
		}
	}
	return false
}

func ipToInt(ip net.IP) *big.Int {
	if v := ip.To4(); v != nil {
		return big.NewInt(0).SetBytes(v)
	}
	return big.NewInt(0).SetBytes(ip.To16())
}
func intToIP(i *big.Int) net.IP {
	return net.IP(i.Bytes())
}

// Return the start and end IP addresses of a given subnet, excluding
// the broadcast address (eg, 192.168.1.255)
func networkRange(ipnet *net.IPNet) (net.IP, net.IP, error) {
	if ipnet.IP == nil {
		return nil, nil, fmt.Errorf("missing field %q in IPAM configuration", "subnet")
	}
	ip, err := canonicalizeIP(ipnet.IP)
	if err != nil {
		return nil, nil, fmt.Errorf("IP not v4 nor v6")
	}

	if len(ip) != len(ipnet.Mask) {
		return nil, nil, fmt.Errorf("IPNet IP and Mask version mismatch")
	}

	var end net.IP
	for i := 0; i < len(ip); i++ {
		end = append(end, ip[i]|^ipnet.Mask[i])
	}

	// Exclude the broadcast address for IPv4
	if ip.To4() != nil {
		end[3]--
	}

	return ipnet.IP, end, nil
}

func canonicalizeIP(ip net.IP) (net.IP, error) {
	if ip.To4() != nil {
		return ip.To4(), nil
	} else if ip.To16() != nil {
		return ip.To16(), nil
	}
	return nil, fmt.Errorf("IP %s not v4 nor v6", ip)
}

// Ensures @ip is within @ipnet, and (if given) inclusive of @start and @end
func validateRangeIP(ip net.IP, ipnet *net.IPNet, start net.IP, end net.IP) error {
	var err error

	// Make sure we can compare IPv4 addresses directly
	ip, err = canonicalizeIP(ip)
	if err != nil {
		return err
	}

	if !ipnet.Contains(ip) {
		return fmt.Errorf("%s not in network: %s", ip, ipnet)
	}

	if start != nil {
		start, err = canonicalizeIP(start)
		if err != nil {
			return err
		}
		if len(ip) != len(start) {
			return fmt.Errorf("%s %d not same size IP address as start %s %d", ip, len(ip), start, len(start))
		}
		for i := 0; i < len(ip); i++ {
			if ip[i] > start[i] {
				break
			} else if ip[i] < start[i] {
				return fmt.Errorf("%s outside of network %s with start %s", ip, ipnet, start)
			}
		}
	}

	if end != nil {
		end, err = canonicalizeIP(end)
		if err != nil {
			return err
		}
		if len(ip) != len(end) {
			return fmt.Errorf("%s %d not same size IP address as end %s %d", ip, len(ip), end, len(end))
		}
		for i := 0; i < len(ip); i++ {
			if ip[i] < end[i] {
				break
			} else if ip[i] > end[i] {
				return fmt.Errorf("%s outside of network %s with end %s", ip, ipnet, end)
			}
		}
	}
	return nil
}

// IPNet like net.IPNet but adds JSON marshalling and unmarshalling
type IPNet net.IPNet

// ParseCIDR takes a string like "10.2.3.1/24" and
// return IPNet with "10.2.3.1" and /24 mask
func ParseCIDR(s string) (*net.IPNet, error) {
	ip, ipn, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}

	ipn.IP = ip
	return ipn, nil
}

func (n *IPNet) String() string{
	return ((*net.IPNet)(n)).String()
}

// MarshalJSON define Marshal
func (n IPNet) MarshalJSON() ([]byte, error) {
	return json.Marshal((*net.IPNet)(&n).String())
}

// UnmarshalJSON define Unmarshal
func (n *IPNet) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	tmp, err := ParseCIDR(s)
	if err != nil {
		return err
	}

	*n = IPNet(*tmp)
	return nil
}

// ReservedIP define
type ReservedIP struct {
	IP net.IP `json:"ip"`
	Gateway net.IP `json:"gateway"`
}