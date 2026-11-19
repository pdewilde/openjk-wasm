package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestProtectedAssetPathsRequireAuth(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "base"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "base", "manifest.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "base", "assets0.pk3"), []byte("pk3"), 0644); err != nil {
		t.Fatal(err)
	}

	h := newStaticHandler(serverConfig{
		dir:      dir,
		authUser: "jedi",
		authPass: "secret",
	}, nil)

	// Public launcher remains accessible without auth.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /, got %d", rec.Code)
	}

	// Direct /base/ requests (including path-normalization variants) require Basic Auth.
	for _, p := range []string{"/base/manifest.json", "/base/assets0.pk3", "//base/assets0.pk3", "/./base/assets0.pk3", "/base/"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for %q, got %d", p, rec.Code)
		}
		if rec.Header().Get("WWW-Authenticate") == "" {
			t.Fatalf("expected WWW-Authenticate header for direct request %q", p)
		}
	}

	// Silent probe suppresses WWW-Authenticate so public visitors do not see a login popup.
	req = httptest.NewRequest(http.MethodGet, "/base/manifest.json", nil)
	req.Header.Set("X-No-Auth-Prompt", "1")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for silent probe, got %d", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("expected no WWW-Authenticate header on X-No-Auth-Prompt probe")
	}

	// Valid credentials succeed even with X-No-Auth-Prompt header set.
	req = httptest.NewRequest(http.MethodGet, "/base/manifest.json", nil)
	req.Header.Set("X-No-Auth-Prompt", "1")
	req.SetBasicAuth("jedi", "secret")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for authenticated /base/manifest.json, got %d", rec.Code)
	}
}

func TestBaseLoginRedirect(t *testing.T) {
	dir := t.TempDir()
	h := newStaticHandler(serverConfig{
		dir:      dir,
		authUser: "jedi",
		authPass: "secret",
	}, nil)

	// Unauthenticated /base/login prompts for Basic Auth.
	req := httptest.NewRequest(http.MethodGet, "/base/login?next=/openjk.wasm32.html", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("expected 401 with WWW-Authenticate on /base/login, got %d", rec.Code)
	}

	// Authenticated /base/login redirects to safe local next path.
	req = httptest.NewRequest(http.MethodGet, "/base/login?next=/openjk.wasm32.html?map=mp/ffa1", nil)
	req.SetBasicAuth("jedi", "secret")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/openjk.wasm32.html?map=mp/ffa1" {
		t.Fatalf("expected 303 redirect to next path, got %d %q", rec.Code, rec.Header().Get("Location"))
	}

	// Unsafe external redirect targets fall back to /.
	for _, bad := range []string{"https://evil.example", "//evil.example", "/\\evil"} {
		req = httptest.NewRequest(http.MethodGet, "/base/login?next="+bad, nil)
		req.SetBasicAuth("jedi", "secret")
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
			t.Fatalf("expected fallback redirect to / for %q, got %d %q", bad, rec.Code, rec.Header().Get("Location"))
		}
	}
}

func TestBaseLogoutCookieForcesUnauthenticatedState(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "base"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "base", "manifest.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	h := newStaticHandler(serverConfig{
		dir:      dir,
		authUser: "jedi",
		authPass: "secret",
	}, nil)

	// Logout sets openjk_hosted_off=1 cookie.
	req := httptest.NewRequest(http.MethodGet, "/base/logout", nil)
	req.Header.Set("X-No-Auth-Prompt", "1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 from silent /base/logout, got %d", rec.Code)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != hostedLoggedOutCookie || cookies[0].Value != "1" {
		t.Fatalf("expected %s=1 cookie from /base/logout, got %+v", hostedLoggedOutCookie, cookies)
	}

	// Even with cached Basic Auth header, openjk_hosted_off=1 forces 401 on /base/manifest.json.
	req = httptest.NewRequest(http.MethodGet, "/base/manifest.json", nil)
	req.SetBasicAuth("jedi", "secret")
	req.AddCookie(cookies[0])
	req.Header.Set("X-No-Auth-Prompt", "1")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when logged-out cookie is present, got %d", rec.Code)
	}

	// /base/login clears the logged-out cookie.
	req = httptest.NewRequest(http.MethodGet, "/base/login?next=/", nil)
	req.SetBasicAuth("jedi", "secret")
	req.AddCookie(cookies[0])
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 from /base/login, got %d", rec.Code)
	}
	cleared := rec.Result().Cookies()
	if len(cleared) == 0 || cleared[0].Name != hostedLoggedOutCookie || cleared[0].MaxAge >= 0 {
		t.Fatalf("expected %s cookie to be cleared by /base/login, got %+v", hostedLoggedOutCookie, cleared)
	}
}
