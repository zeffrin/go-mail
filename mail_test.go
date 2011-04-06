
package mail

import "testing"

var testAddress string = "trangtle@tpg.com.au"

func TestSend(t *testing.T) {
	var m Message;

	m.From.Email = testAddress;
	m.To = make([]Address, 1);
	m.To[0].Name = "Bla";
	m.To[0].Email = "zeffrin@gmail.com";

	m.Body = "Testing";
	
	var s SMTP;

	s.Host = "mail.tpg.com.au";
	s.Port = 25;

	_, err := m.Send(s);

	if err != nil {
		t.Errorf("Send failed: %s", err);
	}
	return;
}
