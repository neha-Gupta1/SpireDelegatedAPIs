package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/spiffe/go-spiffe/v2/logger"
	delegatedidentityv1 "github.com/spiffe/spire-api-sdk/proto/spire/api/agent/delegatedidentity/v1"
	"github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"google.golang.org/grpc"
)

func main() {
	fmt.Println("program staterttt")
	log.Println("program staterttt")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	config := Config{
		AgentAddress: "/run/spire/sockets/agent.sock",
	}
	spiffeSidecar := NewSidecar(&config)
	err := spiffeSidecar.RunDaemon(ctx)
	if err != nil {
		fmt.Println("Error found:***", err)
	}

	select {}
}

// Config contains config variables when creating a SPIFFE Sidecar.
type Config struct {
	AgentAddress string `hcl:"agentAddress"`
	Cmd          string `hcl:"cmd"`
	CmdArgs      string `hcl:"cmdArgs"`
	CertDir      string `hcl:"certDir"`
	// Merge intermediate certificates into Bundle file instead of SVID file,
	// it is useful is some scenarios like MySQL,
	// where this is the expected format for presented certificates and bundles
	AddIntermediatesToBundle bool   `hcl:"addIntermediatesToBundle"`
	SvidFileName             string `hcl:"svidFileName"`
	SvidKeyFileName          string `hcl:"svidKeyFileName"`
	SvidBundleFileName       string `hcl:"svidBundleFileName"`
	RenewSignal              string `hcl:"renewSignal"`
	ReloadExternalProcess    func() error
	Log                      logger.Logger
}

// Sidecar is the component that consumes the Workload API and renews certs
// implements the interface Sidecar
type Sidecar struct {
	config *Config
	// processRunning int32
	// process        *os.Process
	certReadyChan chan struct{}
}

/*
const (
	certsFileMode = os.FileMode(0644)
	keyFileMode   = os.FileMode(0600)
)

// x509Watcher is a sample implementation of the workload.X509SVIDWatcher interface
type x509Watcher struct {
	sidecar *Sidecar
}
*/

// NewSidecar creates a new SPIFFE sidecar
func NewSidecar(config *Config) *Sidecar {
	if config.Log == nil {
		config.Log = logger.Null
	}
	return &Sidecar{
		config:        config,
		certReadyChan: make(chan struct{}, 1),
	}
}

func (s *Sidecar) RunDaemon(ctx context.Context) error {

	SpirePrivilegedAPISocketPath := "/run/spire/sockets-admin/admin.sock"

	sockPath := SpirePrivilegedAPISocketPath
	unixPath := "unix://" + sockPath

	conn, err := grpc.Dial(unixPath, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("spiffe: Cilium grpc.Dial() failed on %s: %s", sockPath, err)
	}

	client := delegatedidentityv1.NewDelegatedIdentityClient(conn)
	// in := delegatedidentityv1.SubscribeToX509BundlesRequest{}
	// client1, err := client.SubscribeToX509Bundles(context.Background(), &in)
	// if err != nil {
	// 	return fmt.Errorf("spiffe: Cilium grpc.Dial() failed on %s: %s", sockPath, err)
	// }
	// response509, err := client1.Recv()
	// if err != nil {
	// 	return fmt.Errorf("spiffe: Cilium grpc.Dial() failed on %s: %s", sockPath, err)
	// }

	// result := response509.GetCaCertificates()
	// fmt.Println("Result: ", result)
	// log.Println("Log Result: ", result)

	req := &delegatedidentityv1.SubscribeToX509SVIDsRequest{
		Selectors: []*types.Selector{
			{Type: "k8s", Value: "ns:default"},
			{Type: "k8s", Value: "pod-name:getcertanother"},
			{Type: "k8s", Value: "pod-name:getcert"},
		}, //getPodSelectors(pod),
	}
	stream, err := client.SubscribeToX509SVIDs(context.Background(), req)
	if err != nil {
		conn.Close()
		return fmt.Errorf("spiffe: Cilium stream failed on %s: %s", sockPath, err)
	}

	response, err := stream.Recv()
	if err != nil {
		conn.Close()
		return fmt.Errorf("spiffe: Cilium stream failed on %s: %s", sockPath, err)
	}
	resp1 := response.String()
	// svids := response.X509Svids
	// stream, err := client.SubscribeToX509SVIDs(context.Background(), req)

	// if err != nil {
	// 	conn.Close()
	// 	return nil, fmt.Errorf("Spiffe: Cilium stream failed on %s: %s", sockPath, err)
	// }

	// svids, err := workloadapi.FetchX509SVIDs(ctx, workloadapi.WithAddr("unix://"+s.config.AgentAddress))
	// if err != nil && !errors.Is(err, context.Canceled) {
	// 	return err
	// }
	// for _, id := range svids {
	log.Println("****id******", resp1)
	// }
	// log.Println("svids here: ", svids)
	// err = workloadapi.WatchX509Context(ctx, &x509Watcher{s}, workloadapi.WithAddr("unix://"+s.config.AgentAddress))
	// if err != nil && !errors.Is(err, context.Canceled) {
	// 	return err
	// }

	return nil
}

/*
// Updates the certificates stored in disk and signal the Process to restart
func (s *Sidecar) updateCertificates(svidResponse *workloadapi.X509Context) {
	s.config.Log.Infof("Updating certificates")

	err := s.dumpBundles(svidResponse)
	if err != nil {
		s.config.Log.Errorf("unable to dump bundle: %v", err)
		return
	}
	err = s.signalProcess()
	if err != nil {
		s.config.Log.Errorf("unable to signal process: %v", err)
	}

	select {
	case s.certReadyChan <- struct{}{}:
	default:
	}
}

// getCmdArgs receives the command line arguments as a string
// and split it at spaces, except when the space is inside quotation marks
func getCmdArgs(args string) ([]string, error) {
	if args == "" {
		return []string{}, nil
	}

	r := csv.NewReader(strings.NewReader(args))
	r.Comma = ' ' // space
	cmdArgs, err := r.Read()
	if err != nil {
		return nil, err
	}

	return cmdArgs, nil
}

// signalProcess sends the configured Renew signal to the process running the proxy
// to reload itself so that the proxy uses the new SVID
func (s *Sidecar) signalProcess() (err error) {
	switch s.config.ReloadExternalProcess {
	case nil:
		if atomic.LoadInt32(&s.processRunning) == 0 {
			cmdArgs, err := getCmdArgs(s.config.CmdArgs)
			if err != nil {
				return fmt.Errorf("error parsing cmd arguments: %w", err)
			}

			cmd := exec.Command(s.config.Cmd, cmdArgs...) // #nosec
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Start()
			if err != nil {
				return fmt.Errorf("error executing process: %v\n%w", s.config.Cmd, err)
			}
			s.process = cmd.Process
			go s.checkProcessExit()
		} else {
			// Signal to reload certs
			sig := unix.SignalNum(s.config.RenewSignal)
			if sig == 0 {
				return fmt.Errorf("error getting signal: %v", s.config.RenewSignal)
			}

			err = s.process.Signal(sig)
			if err != nil {
				return fmt.Errorf("error signaling process with signal: %v\n%w", sig, err)
			}
		}

	default:
		err = s.config.ReloadExternalProcess()
		if err != nil {
			return fmt.Errorf("error reloading external process: %w", err)
		}
	}

	return nil
}

func (s *Sidecar) checkProcessExit() {
	atomic.StoreInt32(&s.processRunning, 1)
	_, err := s.process.Wait()
	if err != nil {
		s.config.Log.Errorf("error waiting for process exit: %v", err)
	}

	atomic.StoreInt32(&s.processRunning, 0)
}

// dumpBundles takes a X509SVIDResponse, representing a svid message from
// the Workload API, and calls writeCerts and writeKey to write to disk
// the svid, key and bundle of certificates.
// It is possible to change output setting `addIntermediatesToBundle` as true.
func (s *Sidecar) dumpBundles(svidResponse *workloadapi.X509Context) error {
	// There may be more than one certificate, but we are interested in the first one only
	svid := svidResponse.DefaultSVID()

	svidFile := path.Join(s.config.CertDir, s.config.SvidFileName)
	svidKeyFile := path.Join(s.config.CertDir, s.config.SvidKeyFileName)
	svidBundleFile := path.Join(s.config.CertDir, s.config.SvidBundleFileName)

	certs := svid.Certificates
	bundleSet, found := svidResponse.Bundles.Get(svid.ID.TrustDomain())
	if !found {
		return fmt.Errorf("no bundles found for %s trust domain", svid.ID.TrustDomain().String())
	}
	bundles := bundleSet.X509Authorities()
	privateKey, err := x509.MarshalPKCS8PrivateKey(svid.PrivateKey)
	if err != nil {
		return err
	}

	// Add intermediates into bundles, and remove them from certs
	if s.config.AddIntermediatesToBundle {
		bundles = append(bundles, certs[1:]...)
		certs = []*x509.Certificate{certs[0]}
	}

	if err := s.writeCerts(svidFile, certs); err != nil {
		return err
	}

	if err := s.writeKey(svidKeyFile, privateKey); err != nil {
		return err
	}

	if err := s.writeCerts(svidBundleFile, bundles); err != nil {
		return err
	}

	return nil
}

// writeCerts takes an array of certificates,
// and encodes them as PEM blocks, writing them to file
func (s *Sidecar) writeCerts(file string, certs []*x509.Certificate) error {
	var pemData []byte
	for _, cert := range certs {
		b := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		}
		pemData = append(pemData, pem.EncodeToMemory(b)...)
	}

	return ioutil.WriteFile(file, pemData, certsFileMode)
}

// writeKey takes a private key as a slice of bytes,
// formats as PEM, and writes it to file
func (s *Sidecar) writeKey(file string, data []byte) error {
	b := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: data,
	}

	return ioutil.WriteFile(file, pem.EncodeToMemory(b), keyFileMode)
}

// OnX509ContextUpdate is run every time an SVID is updated
func (w x509Watcher) OnX509ContextUpdate(svids *workloadapi.X509Context) {
	fmt.Println("**********Recived new svids********")
	fmt.Println("*****svids:", svids.SVIDs)
	for _, svid := range svids.SVIDs {
		fmt.Println("*****svid: ", svid)
		w.sidecar.config.Log.Infof("Received update for spiffeID: %q", svid.ID)
	}

	w.sidecar.updateCertificates(svids)
}

// OnX509ContextWatchError is run when the client runs into an error
func (w x509Watcher) OnX509ContextWatchError(err error) {
	if status.Code(err) != codes.Canceled {
		w.sidecar.config.Log.Errorf("Error while watching x509 context: %v", err)
	}
}

*/
