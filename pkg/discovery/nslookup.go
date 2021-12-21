/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package discovery

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

//
// Implementation of Coherence NSLookup.
// Return pointer to the structure, let caller restrict use through interface.
//

// DiscoveredCluster is a discovered cluster
type DiscoveredCluster struct {
	ClusterName    string
	ConnectionName string
	NSPort         int
	Host           string
	ManagementURLs []string
	SelectedURL    string
	MetricsURLs    []string
	JMXURLs        []string
}

type NSLookup struct {
	Host    string
	Port    int
	channel []byte
	conn    *tcpconn
}

// ClusterNSPort defines the cluster Name and local NS lookup ports
type ClusterNSPort struct {
	HostName    string
	ClusterName string
	Port        int
	IsLocal     bool // indicates this is the cluster owning the NS port queried
}

var (
	multiplexedSocket  = []byte{90, 193, 224, 0} // refer ProtocolIdentifiers
	nameServiceSubPort = []byte{0, 0, 0, 3}
	connectionOpen     = []byte{
		0, 1, 2, 0, 66, 0, 1, 14, 0, 0, 66, 166, 182, 159, 222, 178, 81,
		1, 65, 227, 243, 228, 221, 15, 2, 65, 143, 246, 186, 153, 1, 3,
		65, 248, 180, 229, 242, 4, 4, 65, 196, 254, 220, 245, 5, 5, 65, 215,
		206, 195, 141, 7, 6, 65, 219, 137, 220, 213, 10, 64, 2, 110, 3,
		93, 78, 87, 2, 17, 77, 101, 115, 115, 97, 103, 105, 110, 103, 80,
		114, 111, 116, 111, 99, 111, 108, 2, 65, 2, 65, 2, 19, 78, 97, 109,
		101, 83, 101, 114, 118, 105, 99, 101, 80, 114, 111, 116, 111, 99,
		111, 108, 2, 65, 1, 65, 1, 5, 160, 2, 0, 0, 14, 0, 0, 66, 174, 137,
		158, 222, 178, 81, 1, 65, 129, 128, 128, 240, 15, 5, 65, 152, 159,
		129, 128, 8, 6, 65, 147, 158, 1, 64, 1, 106, 2, 110, 3, 106, 4, 113,
		5, 113, 6, 78, 8, 67, 108, 117, 115, 116, 101, 114, 66, 9, 78, 9, 108,
		111, 99, 97, 108, 104, 111, 115, 116, 10, 78, 5, 50, 48, 50, 51, 51, 12,
		78, 16, 67, 111, 104, 101, 114, 101, 110, 99, 101, 67, 111, 110, 115,
		111, 108, 101, 64, 64,
	}
	channelOpen = []byte{
		0, 11, 2, 0, 66, 1, 1, 78, 19, 78, 97, 109, 101, 83, 101, 114, 118,
		105, 99, 101, 80, 114, 111, 116, 111, 99, 111, 108, 2, 78, 11, 78,
		97, 109, 101, 83, 101, 114, 118, 105, 99, 101, 64,
	}
	nsLookupReqID = []byte{
		1, 1, 0, 66, 0, 1, 78,
	}
	reqEndMarker = []byte{
		64,
	}
)

const DefaultPort = 7574
const ClusterNameLookup = "Cluster/name"
const ClusterInfoLookup = "Cluster/info"
const ClusterForeignLookup = "Cluster/foreign"
const ManagementLookup = "management/HTTPManagementURL"
const JMXLookup = "management/JMXServiceURL"
const MetricsLookup = "metrics/HTTPMetricsURL"
const NSPrefix = "NameService/string/"
const NSLocalPort = "/NameService/localPort"

// internal net.TCPConn wrapper
type tcpconn struct {
	*net.TCPConn
}

// Open returns a NSLookup instance which represents a connection to the NameService of a Coherence
// cluster, identified by an internal Channel ID
func Open(hostPort string, timeout int32) (*NSLookup, error) {
	var (
		nsLookup = NSLookup{}
		err      error
		host     string
		port     int
	)

	// Default to look at localhost:7574
	if hostPort == "" {
		nsLookup.Host = "localhost"
		nsLookup.Port = DefaultPort
	} else {
		// parse the host/ports to make sure they are valid
		hostIP := strings.Split(hostPort, ":")
		if len(hostIP) == 1 {
			// default to DefaultPort (7574)
			nsLookup.Host = hostIP[0]
			nsLookup.Port = DefaultPort
		} else {
			if len(hostIP) != 2 {
				return &nsLookup, errors.New("invalid value for host/port of [" + hostPort + "]")
			}

			host = hostIP[0]

			port, err = strconv.Atoi(hostIP[1])
			if err != nil {
				return &nsLookup, errors.New("invalid port value of [" + hostIP[1] + "]")
			}

			if port < 1024 || port > 65535 {
				return &nsLookup, fmt.Errorf("value for port of %d is invalid", port)
			}
			nsLookup.Host = host
			nsLookup.Port = port
		}
	}

	err = nsLookup.connect(nsLookup.getAddress(), timeout)
	if err != nil {
		return nil, err
	}

	return &nsLookup, nil
}

// GetHost returns the host
func (n *NSLookup) GetHost() string {
	return n.Host
}

// GetPort returns the host
func (n *NSLookup) GetPort() int {
	return n.Port
}

// DiscoverClusterInfo discovers cluster information for the specific NS
// this method assumes that we are only interested in the local information as
// foreign clusters will have already had their ephemeral NS port retrieved
func (n *NSLookup) DiscoverClusterInfo() (DiscoveredCluster, error) {
	var (
		err     error
		value   string
		cluster = DiscoveredCluster{}
	)

	cluster.NSPort = n.Port
	cluster.Host = n.Host

	// get the cluster name
	cluster.ClusterName, err = n.Lookup(ClusterNameLookup)
	if err != nil {
		return cluster, err
	}

	// management lookup
	value, err = n.Lookup(NSPrefix + ManagementLookup)
	if err != nil {
		return cluster, err
	}

	cluster.ManagementURLs = parseResults(value)

	// JMX lookup
	value, err = n.Lookup(JMXLookup)
	if err != nil {
		return cluster, err
	}

	cluster.JMXURLs = parseResults(value)

	// JMX lookup
	value, err = n.Lookup(NSPrefix + MetricsLookup)
	if err != nil {
		return cluster, err
	}

	cluster.MetricsURLs = parseResults(value)

	return cluster, nil
}

// DiscoverNameServicePorts discovers any clusters bound to the NS port and returns a struct
// containing each of the ports and the clusters
func (n *NSLookup) DiscoverNameServicePorts() ([]ClusterNSPort, error) {
	var (
		clusterNames  = make([]string, 0)
		err           error
		port          string
		localCluster  string
		otherClusters string
		othersList    []string
		clusterCount  int
		i             = 1
		intPort       int
	)

	localCluster, err = n.Lookup(ClusterNameLookup)
	if err != nil {
		return nil, err
	}

	otherClusters, err = n.Lookup(NSPrefix + ClusterForeignLookup)
	if err != nil {
		return nil, err
	}

	// foreign clusters are in the format "[cluster1, cluster2, clusterN]"
	othersList = parseResults(otherClusters)
	if len(othersList) > 0 {
		clusterNames = append(clusterNames, othersList...)
	}

	clusterCount = len(clusterNames)
	listClusters := make([]ClusterNSPort, clusterCount+1)

	// add the local cluster first
	listClusters[0] = ClusterNSPort{ClusterName: localCluster, Port: n.Port, IsLocal: true, HostName: n.Host}

	if clusterCount != 0 {
		for _, cluster := range clusterNames {
			// lookup the local NS port
			port, err = n.Lookup(NSPrefix + ClusterForeignLookup + "/" + cluster + NSLocalPort)
			if err != nil {
				return nil, err
			}
			intPort, err = strconv.Atoi(port)
			if err != nil {
				return nil, err
			}
			listClusters[i] = ClusterNSPort{ClusterName: cluster, Port: intPort, HostName: n.Host}
			i++
		}
	}

	return listClusters, nil
}

// Lookup looks up a name
func (n *NSLookup) Lookup(name string) (string, error) {
	bName, err := n.lookupInternal(name)
	if err != nil {
		return "", err
	}

	if len(bName) <= 7 {
		return "", nil
	}

	return n.conn.readString(bName), nil
}

// lookupInternal raw lookup (returns byte array) on a string
func (n *NSLookup) lookupInternal(name string) ([]byte, error) {
	request := make([]byte, 0)
	request = append(request, n.channel...)
	request = append(request, nsLookupReqID...)

	writer := bytes.NewBuffer(make([]byte, 0))
	err := writePackedInt(writer, len(name))
	if err != nil {
		return nil, err
	}

	request = append(request, writer.Bytes()...)
	request = append(request, name...)
	request = append(request, reqEndMarker...)

	err = writePackedInt(n.conn, len(request))
	if err != nil {
		return nil, err
	}
	_, err = n.conn.Write(request)
	if err != nil {
		return nil, err
	}

	// read the response
	response, err := n.conn.read()
	if err != nil {
		return nil, err
	}

	return response[len(n.channel)+1:], nil // strip channel id and request id from the response
}

// Close closes the connection
func (n *NSLookup) Close() error {
	return n.conn.Close()
}

// connect establishes a TCP connection with the cluster's port and subport,
// and opens a connection and channel
func (n *NSLookup) connect(address string, timeout int32) error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return err
	}

	c, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return fmt.Errorf("unable to connect to %s: %s", address, err.Error())
	}
	n.conn = &tcpconn{c}

	// set timeout for read and write
	err = n.conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
	if err != nil {
		return err
	}
	err = n.conn.SetWriteDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
	if err != nil {
		return err
	}

	err = n.conn.SetKeepAlive(true)
	if err != nil {
		return err
	}

	if err = n.conn.SetNoDelay(true); err != nil {
		return err
	}

	_, err = n.conn.Write(multiplexedSocket)
	if err != nil {
		return err
	}

	_, err = n.conn.Write(nameServiceSubPort)
	if err != nil {
		return err
	}

	err = writePackedInt(n.conn, len(connectionOpen))
	if err != nil {
		return err
	}
	_, err = n.conn.Write(connectionOpen)
	if err != nil {
		return err
	}

	err = writePackedInt(n.conn, len(channelOpen))
	if err != nil {
		return err
	}
	_, err = n.conn.Write(channelOpen)
	if err != nil {
		return err
	}

	// wait for and skip over response to connect request
	_, err = n.conn.read()
	if err != nil {
		return err
	}

	// read the channel ID
	data, err := n.conn.read()
	n.channel = data[8 : 8+len(data)-9]
	if err != nil {
		return err
	}

	return nil
}

// writePackedInt writes packed int on specified writer
func writePackedInt(w io.Writer, n int) error {
	var (
		b   = 0
		err error
	)

	if n < 0 {
		b = 0x40
		n = ^n
	}

	b |= n & 0x3F
	n >>= 6

	for n != 0 {
		b |= 0x80
		_, err = w.Write([]byte{byte(b)})
		if err != nil {
			return err
		}

		b = n & 0x7f
		n >>= 7
	}
	_, err = w.Write([]byte{byte(b)})

	return err
}

// read reads and processes response
func (conn *tcpconn) read() ([]byte, error) {
	c, _, err := conn.readPackedInt(conn)
	if err != nil {
		return nil, err
	}

	if c < 0 {
		return nil, errors.New("received a message with a negative length")
	} else if c == 0 {
		return nil, errors.New("received a message with a length of zero")
	} else {
		data := make([]byte, c)
		_, err := conn.Read(data)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
}

// readPackedInt reads a packed int
func (conn *tcpconn) readPackedInt(r io.Reader) (int, int, error) {
	var (
		msg      = "unable to read packed int"
		data1    = make([]byte, 1)
		negative bool
		bits     = 6
	)

	pos := 0

	_, err := r.Read(data1)
	if err != nil {
		return 0, 0, errors.New(msg + ": " + err.Error())
	}
	value := data1[0]
	var n = int(value & 0x3F) // 6 bits of data in first byte

	if (value & 0x40) != 0 {
		negative = true
	} else {
		negative = false
	}

	for {
		if (value & 0x80) != 0 {
			pos++
			_, err = r.Read(data1)
			if err != nil {
				return 0, 0, errors.New(msg + ": " + err.Error())
			}
			value = data1[0]
			n |= int(int(value&0x7F) << bits)
			bits += 7
		} else {
			break
		}
	}

	if negative {
		n = ^n
	}

	return n, pos, nil
}

// readString strips channel Id / response number and converts to string
func (conn *tcpconn) readString(b []byte) string {
	resultLen, pos, _ := conn.readPackedInt(bytes.NewReader(b[6:]))

	return string(b[7+pos : resultLen+7+pos])
}

// getAddress returns the host:port address
func (n *NSLookup) getAddress() string {
	return fmt.Sprintf("%s:%d", n.Host, n.Port)
}

// parseResults parses a string in the format "[value, value, value]" and returns as an array
func parseResults(results string) []string {
	result := strings.ReplaceAll(results, "[", "")
	result = strings.ReplaceAll(result, "]", "")
	if result != "" {
		return strings.Split(result, ", ")
	}

	return []string{}
}
