// Copyright (c) 2017 Kevin Browder
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/jawher/mow.cli"
	"github.com/miekg/dns"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"
)

func UpdateRoute53(ipAddress net.IP, hostZoneId string, name string) {
	svc := route53.New(session.New())
	request := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(name),
						Type: aws.String("A"),
						TTL:  aws.Int64(300),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(ipAddress.String()),
							},
						},
					},
				},
			},
			Comment: aws.String(fmt.Sprintf("Updating %s at %s", name, time.Now())),
		},
		HostedZoneId: aws.String(hostZoneId),
	}
	resp, err := svc.ChangeResourceRecordSets(request)
	if err != nil {
		panic(err)
	}

	fmt.Println("Change Response:")
	fmt.Println(resp)
}

func getExternalIP() net.IP {
	client := new(dns.Client)

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn("myip.opendns.com"), dns.TypeA)

	r, _, err := client.Exchange(m, net.JoinHostPort("resolver1.opendns.com", "53"))

	if r == nil {
		panic(err)
	}

	if r.Rcode != dns.RcodeSuccess {
		panic(err)
	}
	if t, ok := r.Answer[0].(*dns.A); ok {
		myIp := t.A
		return myIp
	}
	panic("Couldn't get an IP")
}

func main() {
	app := cli.App("r53ddns", "Update Route53 with our current dynamic dns address")

	//app.Spec = "ZoneId Domain"
	var (
		zoneId = app.StringArg("ZONEID", "", "Zone id in route53")
		domain = app.StringArg("DOMAIN", "", "Domain name to update (usually A record)")
	)

	app.Action = func() {
		fmt.Printf("arg1: %s, %s\n", *zoneId, *domain)
		myIp := getExternalIP()

		fmt.Printf("External IP Addresss: %v\n", myIp)
		domainName := *domain
		lastUpdateFile := fmt.Sprintf(".lastupdate.%s", domainName)
		if _, err := os.Stat(lastUpdateFile); os.IsNotExist(err) {
			ioutil.WriteFile(lastUpdateFile, []byte(""), 0600)
		}
		data, err := ioutil.ReadFile(lastUpdateFile)
		if err != nil {
			panic(err)
		}
		lastIpAddress := net.ParseIP(strings.TrimSpace(string(data)))
		if !(myIp.Equal(lastIpAddress)) {
			fmt.Printf("IP Changed, updating route53\n")
			UpdateRoute53(myIp, *zoneId, domainName)
			ioutil.WriteFile(lastUpdateFile, []byte(myIp.String()), 0600)
		} else {
			fmt.Printf("IP Unchanged\n")
		}
	}

	app.Run(os.Args)
}
