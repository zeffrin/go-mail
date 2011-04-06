/*
	Package for connecting to an SMTP host and sending an email
*/


package mail

import "os"
import "net"
import "bufio"
import "strings"
import "fmt"
import "strconv"

// Type for containing Email address information, Name string and the user@domain.com address
type Address struct {
	Name	string;
	Email	string;
}

// Type for describing an email Message
// From and To are Address structs
// Body is simply a string containing the text body of the email
// Attachments is a slice which may be loaded with Attachment references
type Message struct {
	From		Address;
	To		[]Address;
	CC		[]Address;
	BCC		[]Address;
	Body		string;
	Attachments	[]Attachment;
}

// Type for storing attachments and their related header info
type Attachment struct {
	ContentID		string;
	ContentType		string;
	TransferEncoding	int;
	Data			[]byte;
}

// Type for defining SMTP server parameters
type SMTP struct {
	Host	string;
	Port	int;
}

// Internally used linked list for getting text responses from smtp

type response struct {
	text	string;
	next	*response;
}

type readwrite struct {
	r	*bufio.Reader;
	w	*bufio.Writer;
}

var (
	ErrInvalidFromValue	= os.NewError("from email undefined");
	ErrNoRecipients		= os.NewError("no recipients defined");
	ErrInvalidToValue	= os.NewError("to email undefined");
	ErrBodyUndefined	= os.NewError("body undefined");
	ErrHostUndefined	= os.NewError("host undefined");
	ErrPortUndefined	= os.NewError("port undefined");
	ErrNotReady		= os.NewError("server reports not ready");
	ErrHeloNotAccepted	= os.NewError("EHLO/HELO not accepted");
	ErrDataNotAccepted	= os.NewError("DATA not accepted");
	ErrQuitNotAccepted	= os.NewError("QUIT not accepted");
)


// Function to action delivery of the message
func (m *Message) Send(host SMTP) (n int, err os.Error) {

	// Check we have atleast the minimum details to send a message
	switch {

	case len(strings.Split(m.From.Email, "@", 0)) != 2:
		err = ErrInvalidFromValue;
		return;

	case len(m.To) == 0:
		err = ErrNoRecipients;
		return;

	/* TODO Will want to validate addresses
	case len(strings.Split(m.To.Email, "@", 0)) != 2:
		err = ErrInvalidToValue;
		return;
	*/

	case len(m.Body) == 0:
		err = ErrBodyUndefined;
		return;

	case len(host.Host) == 0:
		err = ErrHostUndefined;
		return;

	case host.Port == 0:
		err = ErrPortUndefined;
		return;
	}

	// Connect to SMTP server
	conn, err := net.Dial("tcp", "", fmt.Sprintf("%s:%d", host.Host, host.Port));
	if err != nil {
		return
	}

	// set up a writer to output to the socket stream
	// unsure, is it really necessary to use bufio.NewWriter here?
	var rw readwrite;
	rw.w = bufio.NewWriter(conn);
	rw.r = bufio.NewReader(conn);

	// ensure server is ready
	var rcode int;
	var rtext *response;

	rcode, rtext, err = dorecv(rw);
	if err != nil {
		return
	}

	if rcode != 220 {
		err = ErrNotReady;
		return;
	}

	// Introduce ourselves to the server using EHLO and the domain portion of our from address
	rcode, rtext, err = sendrecv(rw, fmt.Sprintf("EHLO %s\r\n", strings.Split(m.From.Email, "@", 2)[1]));
	if err != nil {
		return
	}

	// If EHLO wasn't accepted we'll try the older HELO introduction
	if rcode != 250 {
		rcode, rtext, err = sendrecv(rw, fmt.Sprintf("HELO %s\r\n", strings.Split(m.From.Email, "@", 2)[1]));
		if err != nil {
			return
		}

		if rcode != 250 {
			err = ErrHeloNotAccepted;
			return;
		}

	}

	// Need to perform one mail transaction for To and CC,
	// then one each per BCC recipient
	for i, j := 0, 1+len(m.BCC); i < j; i++ {
		// Start the mail transaction
		if len(m.From.Name) == 0 {
			rcode, rtext, err = sendrecv(rw, fmt.Sprintf("MAIL FROM:<%s>\r\n", m.From.Email))
		} else {
			rcode, rtext, err = sendrecv(rw, fmt.Sprintf("MAIL FROM:\"%s\"<%s>\r\n", m.From.Name, m.From.Email))
		}
		if err != nil {
			return
		}

		if rcode != 250 {
			err = ErrInvalidFromValue;
			return;
		}

		if i >= len(m.BCC) {

			// Add normal recipients
			for k, l := 0, len(m.To); k < l; k++ {
				if len(m.To[k].Name) > 0 {
					rcode, rtext, err = sendrecv(rw, fmt.Sprintf("RCPT TO:\"%s\"<%s>\r\n", m.To[k].Name, m.To[k].Email))
				} else {
					rcode, rtext, err = sendrecv(rw, fmt.Sprintf("RCPT TO:<%s>\r\n", m.To[k].Email))
				}
				if err != nil {
					return
				}

				if rcode != 250 {
					err = ErrInvalidToValue;
					return;
				}
			}

			// TODO Find some neater way of doing both in one loop

			for k, l := 0, len(m.CC); k < l; k++ {
				if len(m.To[k].Name) > 0 {
					rcode, rtext, err = sendrecv(rw, fmt.Sprintf("RCPT TO:\"%s\"<%s>\r\n", m.To[k].Name, m.To[k].Email))
				} else {
					rcode, rtext, err = sendrecv(rw, fmt.Sprintf("RCPT TO:<%s>\r\n", m.To[k].Email))
				}
				if err != nil {
					return
				}

				if rcode != 250 {
					err = ErrInvalidToValue;
					return;
				}
			}

			rcode, rtext, err = sendrecv(rw, "DATA\r\n");
			if err != nil {
				return
			}
			if rcode != 354 {
				err = ErrDataNotAccepted;
				return;
			}

			rcode, rtext, err = sendrecv(rw, ".\r\n");
			if err != nil {
				return
			}

			// TODO send header information
			// TODO send message body finishing with CRLF.CRLF

		} else {
			// send to the BCC recipients with modified header
		}


		// TODO send attachments

		// send QUIT and wait for reply to close connection
		rcode, rtext, err = sendrecv(rw, "QUIT\r\n");
		if err != nil {
			return
		}
		if rcode != 221 {
			err = ErrQuitNotAccepted;
			return;
		}

	}	// end for


	// just to get rid of a compiler error for not using it
	// should really return it in case of err
	_ = fmt.Sprintf("%#v", rtext);

	err = conn.Close();

	return;
}

// Send a message to the Writer and get response from Reader
// rtext *response is a linked list containing the textual responses
func sendrecv(rw readwrite, m string) (rcode int, rtext *response, err os.Error) {
	err = dosend(rw, m);
	if err != nil {
		return
	}

	rcode, rtext, err = dorecv(rw);
	return;
}

func dorecv(rw readwrite) (rcode int, rtext *response, err os.Error) {
	done := false;
	var buf string;
	var curr *response;

	// setting up a counter though ending loop is based on done variable
	for i := 0; done == false; i++ {
		buf, err = rw.r.ReadString('\n');
		if err != nil {
			return
		}

		if len(buf) > 3 && buf[3] != 45 {
			done = true
		}

		fmt.Printf("%s\n", buf);

		if i == 0 {
			rtext = new(response);
			rtext.text = buf[4:];
			curr = rtext;
		} else {
			curr.next = new(response);
			*curr = *curr.next;
			curr.text = buf[4:];
		}

	}
	rcode, err = strconv.Atoi(buf[0:3]);
	return;
}

func dosend(rw readwrite, m string) (err os.Error) {
	fmt.Printf("SEND: %s\n", m);

	rw.w.WriteString(m);
	if err != nil {
		return
	}

	rw.w.Flush();
	return;
}
