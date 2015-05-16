package main

import (
	"testing"
	"github.com/miekg/dns"
	. "github.com/smartystreets/goconvey/convey"
)

/*
	WIP, nothing interesting here.
	This is just a stub for a test playground.
*/
func TestBlacklistedDomains(t *testing.T) {
	letters := []string{"seznam.cz", "youtube.com", "root.cz", "google.com", "karms.biz"}

	Convey("Bla bla bla", t, func() {
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(letters[0]), dns.TypeMX)
		m.Id = dns.Id()
		m.RecursionDesired = true
		c := new(dns.Client)

		r, _, err := c.Exchange(m, "127.0.0.1:53535")
		Convey("Sinked", func() {
			So(err, ShouldBeNil)
			// IP address of the sinkhole....
			So(r.Answer, ShouldContain, "127.0.0.1")
		})
	})
}

