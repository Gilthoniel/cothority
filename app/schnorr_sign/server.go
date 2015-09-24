package schnorr_sign

import log "github.com/Sirupsen/logrus"
import "github.com/dedis/cothority/deploy"
import "github.com/dedis/cothority/lib/config"
import "github.com/dedis/crypto/poly"
import dbg "github.com/dedis/cothority/lib/debug_lvl"

func RunServer(hosts *config.HostsConfig, app *config.AppConfig, depl *deploy.Config) {
	s := config.GetSuite(depl.Suite)
	poly.SUITE = s
	poly.SECURITY = poly.MODERATE
	n := len(hosts.Hosts)

	info := poly.PolyInfo{
		N: n,
		R: n,
		T: n,
	}
	indexPeer := -1
	for i, h := range hosts.Hosts {
		if h == app.Hostname {
			indexPeer = i
			break
		}
	}
	if indexPeer == -1 {
		log.Fatal("Peer ", app.Hostname, "(", app.PhysAddr, ") did not find any match for its name.Abort")
	}

	dbg.Lvl2("Creating new peer ", app.Hostname, "(", app.PhysAddr, ") ...")
	// indexPeer == 0 <==> peer is root
	p := NewPeer(indexPeer, app.Hostname, info, indexPeer == 0)

	// make it listen
	dbg.Lvl2("Peer ", app.Hostname, "is now listening for incoming connections")
	go p.Listen()

	// then connect it to its successor in the list
	for _, h := range hosts.Hosts[indexPeer+1:] {
		dbg.Lvl2("Peer ", app.Hostname, " will connect to ", h)
		// will connect and SYN with the remote peer
		p.ConnectTo(h)
	}
	// Wait until this peer is connected / SYN'd with each other peer
	p.WaitSYNs()

	// start to record
	t1 := time.Now()

	// Setup the schnorr system amongst peers
	p.SetupDistributedSchnorr()
	p.SendACKs()
	p.WaitACKs()
	dbg.Lvl1(p.String(), "completed Schnorr setup")

	// send setup time
	delta := time.Since(t1)
	dbt.Lvl2(p.String(), "setup accomplished in ", delta, " sec")
	//log.WithFields(log.Fields{
	//	"file":  logutils.File(),
	//	"type":  "root_round",
	//	"round": 0,
	//	"time":  delta,
	//}).Info("")

	// Then issue a signature !
	t2 := time.Now()
	msg := "hello world"
	sig := p.SchnorrSig([]byte(msg))
	err := p.VerifySchnorrSig(sig, []byte(msg))
	if err != nil {
		dbg.Fatal(p.String(), "could not verify schnorr signature :/ ", err)
	}
	//arr := p.BroadcastSignature(sig)
	//for i, _ := range arr {
	//	err := p.VerifySchnorrSig(arr[i], []byte(msg))
	//	if err != nil {
	//		dbg.Fatal(p.String(), "could not verify issued schnorr signature : ", err)
	//	}
	//}
	dbg.Lvl1(p.String(), "verified the schnorr sig !")
	// record time
	delta2 := time.Since(t2)
	dbg.Lvl2(p.String(), "signature done in ", delta2, "secs")
	log.WithFields(log.Fields{
		"file":  logutils.File(),
		"type":  "root_round",
		"round": 0,
		"time":  delta,
	}).Info("")

	p.WaitFins()
	dbg.Lvl1(p.String(), "is leaving ...")
}
