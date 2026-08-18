package zigbee

import "testing"

func TestCorrelatorResolvesOK(t *testing.T) {
	c := NewResponseCorrelator()
	tx, ch := c.NewTransaction()

	c.HandleResponse([]byte(`{"data":{"from":"old","to":"new"},"status":"ok","transaction":"` + tx + `"}`))

	select {
	case res := <-ch:
		if !res.OK {
			t.Errorf("expected OK result, got error %q", res.Error)
		}
	default:
		t.Fatal("expected result on channel")
	}
}

func TestCorrelatorResolvesError(t *testing.T) {
	c := NewResponseCorrelator()
	tx, ch := c.NewTransaction()

	c.HandleResponse([]byte(`{"data":{},"status":"error","error":"name already in use","transaction":"` + tx + `"}`))

	select {
	case res := <-ch:
		if res.OK {
			t.Error("expected error result")
		}
		if res.Error != "name already in use" {
			t.Errorf("unexpected error text: %q", res.Error)
		}
	default:
		t.Fatal("expected result on channel")
	}
}

func TestCorrelatorIgnoresUnknownTransaction(t *testing.T) {
	c := NewResponseCorrelator()
	_, ch := c.NewTransaction()

	c.HandleResponse([]byte(`{"status":"ok","transaction":"someone-else"}`))
	c.HandleResponse([]byte(`{"status":"ok"}`))
	c.HandleResponse([]byte(`not json`))

	select {
	case <-ch:
		t.Error("expected no result for unrelated responses")
	default:
	}
}

func TestCorrelatorForget(t *testing.T) {
	c := NewResponseCorrelator()
	tx, ch := c.NewTransaction()
	c.Forget(tx)

	c.HandleResponse([]byte(`{"status":"ok","transaction":"` + tx + `"}`))

	select {
	case <-ch:
		t.Error("expected no result after Forget")
	default:
	}
}
