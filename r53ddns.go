
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
	"github.com/miekg/dns"
	"io/ioutil"
	"log"
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

func main() {
	client := new(dns.Client)

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn("myip.opendns.com"), dns.TypeA)

	r, _, err := client.Exchange(m, net.JoinHostPort("resolver1.opendns.com", "53"))

	if r == nil {
		log.Fatalf("*** error: %s\n", err.Error())
	}

	if r.Rcode != dns.RcodeSuccess {
		log.Fatalf(" *** invalid answer name %s after A query for %s\n", os.Args[1], os.Args[1])
	}

	if t, ok := r.Answer[0].(*dns.A); ok {
		myIp := t.A
		fmt.Printf("External IP Addresss: %v\n", myIp)
		lastUpdateFile := ".lastupdate"
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
			UpdateRoute53(myIp, os.Args[1], os.Args[2])
			ioutil.WriteFile(lastUpdateFile, []byte(myIp.String()), 0600)
		} else {
			fmt.Printf("IP Unchanged\n")
		}

		return
	}
}
