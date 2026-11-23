package harness

import (
	"fmt"
	"os/exec"
	"strings"
)

// SQL runs statements through psql in one session on this session's side's
// PostgreSQL (condition C2) and records them with their output, or with the
// ERROR line when a statement fails, as one note. Each statement is a
// separate -c, so BEGIN … COMMIT spans several of them. Notices are off:
// they depend on leftovers of earlier runs (IF NOT EXISTS).
func (s *Session) SQL(stmts ...string) {
	s.T.Helper()
	args := []string{"compose", "-f", envOr("COMPAT_COMPOSE_FILE", "../docker-compose.yml"), "--profile", Profile(),
		"exec", "-T", "-e", "PGOPTIONS=-c client_min_messages=warning", serviceFor("postgres", s.Target.Name), "psql", "-X", "-q", "-tA", "-v", "ON_ERROR_STOP=1", "-U", "compat", "-d", "compat"}
	for _, st := range stmts {
		args = append(args, "-c", st)
	}
	out, err := exec.Command("docker", args...).CombinedOutput() //nolint:gosec // fixed binary, args from scenario code
	res := strings.TrimSpace(string(out))
	if err != nil {
		var e []string
		for _, l := range strings.Split(res, "\n") {
			if strings.HasPrefix(l, "ERROR:") {
				e = append(e, strings.Join(strings.Fields(l), " "))
			}
		}
		if len(e) == 0 {
			s.T.Fatalf("%s: psql: %v\n%s", s.Target.Name, err, out)
		}
		res = strings.Join(e, "; ")
	}
	s.Note(fmt.Sprintf("sql %s -> %s", strings.Join(stmts, "; "), res))
}

// probePostgres creates a table through each side's PostgreSQL and expects
// the other side not to have it.
func probePostgres() error {
	psql := func(side, sql string) (string, error) {
		out, err := compose("exec", "-T", "postgres-"+side, "psql", "-U", "compat", "-d", "compat", "-tAc", sql)
		return strings.TrimSpace(out), err
	}
	for _, pair := range [][2]string{{"ref", "sut"}, {"sut", "ref"}} {
		table := "compat_iso_probe_" + pair[0]
		if _, err := psql(pair[0], "CREATE TABLE "+table+" (k int)"); err != nil {
			return err
		}
		own, err := psql(pair[0], "SELECT to_regclass('"+table+"') IS NOT NULL")
		if err != nil {
			return err
		}
		other, err := psql(pair[1], "SELECT to_regclass('"+table+"') IS NOT NULL")
		if err != nil {
			return err
		}
		if own != "t" || other != "f" {
			return fmt.Errorf("CS-ISO-01: table created via postgres-%s exists %s there, %s via postgres-%s (want t, f)", pair[0], own, other, pair[1])
		}
		if _, err := psql(pair[0], "DROP TABLE "+table); err != nil {
			return err
		}
	}
	return nil
}
