package controller

import (
	"bytes"
	"strings"
	"time"

	"github.com/tidwall/resp"
	"github.com/tidwall/tile38/internal/glob"
	"github.com/tidwall/tile38/internal/server"
)

func (c *Controller) cmdKeys(msg *server.Message) (res resp.Value, err error) {
	var start = time.Now()
	vs := msg.Values[1:]

	var pattern string
	var ok bool
	if vs, pattern, ok = tokenval(vs); !ok || pattern == "" {
		return server.NOMessage, errInvalidNumberOfArguments
	}
	if len(vs) != 0 {
		return server.NOMessage, errInvalidNumberOfArguments
	}

	var wr = &bytes.Buffer{}
	var once bool
	if msg.OutputType == server.JSON {
		wr.WriteString(`{"ok":true,"keys":[`)
	}
	var everything bool
	var greater bool
	var greaterPivot string
	var vals []resp.Value

	iterator := func(key string, value interface{}) bool {
		var match bool
		if everything {
			match = true
		} else if greater {
			if !strings.HasPrefix(key, greaterPivot) {
				return false
			}
			match = true
		} else {
			match, _ = glob.Match(pattern, key)
		}
		if match {
			if once {
				if msg.OutputType == server.JSON {
					wr.WriteByte(',')
				}
			} else {
				once = true
			}
			switch msg.OutputType {
			case server.JSON:
				wr.WriteString(jsonString(key))
			case server.RESP:
				vals = append(vals, resp.StringValue(key))
			}
		}
		return true
	}
	if pattern == "*" {
		everything = true
		c.cols.Scan(iterator)
	} else {
		if strings.HasSuffix(pattern, "*") {
			greaterPivot = pattern[:len(pattern)-1]
			if glob.IsGlob(greaterPivot) {
				greater = false
				c.cols.Scan(iterator)
			} else {
				greater = true
				c.cols.Ascend(greaterPivot, iterator)
			}
		} else if glob.IsGlob(pattern) {
			greater = false
			c.cols.Scan(iterator)
		} else {
			greater = true
			greaterPivot = pattern
			c.cols.Ascend(greaterPivot, iterator)
		}
	}
	if msg.OutputType == server.JSON {
		wr.WriteString(`],"elapsed":"` + time.Now().Sub(start).String() + "\"}")
		return resp.StringValue(wr.String()), nil
	}
	return resp.ArrayValue(vals), nil
}
