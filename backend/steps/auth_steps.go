package steps

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cucumber/godog"

	"authgate/internal/auth"
)

func registerAuthSteps(sc *godog.ScenarioContext) {
	sc.Step(`^I log in as "([^"]*)" with password "([^"]*)"$`, logIn)
	sc.Step(`^I am logged in as "([^"]*)" with password "([^"]*)"$`, alreadyLoggedIn)
	sc.Step(`^I fail to log in as "([^"]*)" (\d+) times$`, failLogins)
	sc.Step(`^the login is "(successful|rejected)"$`, loginIs)
	sc.Step(`^the login failure reason is "([^"]*)"$`, loginReason)
	sc.Step(`^I receive a session token$`, haveToken)
	sc.Step(`^I log out$`, logOut)
	sc.Step(`^(\d+) minutes? pass(?:es)?$`, minutesPass)
	sc.Step(`^my session is "(valid|invalid)"$`, sessionIs)
}

func logIn(ctx context.Context, user, pass string) error {
	w := from(ctx)
	w.sess, w.loginErr = w.auth.Login(user, pass)
	w.loggedIn = true
	return nil
}

func alreadyLoggedIn(ctx context.Context, user, pass string) error {
	_ = logIn(ctx, user, pass)
	if err := from(ctx).loginErr; err != nil {
		return fmt.Errorf("precondition failed: could not log in as %s: %v", user, err)
	}
	return nil
}

func failLogins(ctx context.Context, user string, n int) error {
	w := from(ctx)
	for i := 0; i < n; i++ {
		if _, err := w.auth.Login(user, fmt.Sprintf("wrong-password-%d", i)); err == nil {
			return fmt.Errorf("login unexpectedly succeeded on attempt %d", i+1)
		}
	}
	return nil
}

func loginIs(ctx context.Context, want string) error {
	w := from(ctx)
	if !w.loggedIn {
		return errors.New("no login was attempted")
	}
	got := "successful"
	if w.loginErr != nil {
		got = "rejected"
	}
	if got != want {
		return fmt.Errorf("expected login to be %s but it was %s (%v)", want, got, w.loginErr)
	}
	return nil
}

func loginReason(ctx context.Context, want string) error {
	var le *auth.LoginError
	if err := from(ctx).loginErr; !errors.As(err, &le) {
		return fmt.Errorf("expected failure reason %q but login did not fail (err=%v)", want, err)
	} else if le.Reason != want {
		return fmt.Errorf("expected failure reason %q but got %q", want, le.Reason)
	}
	return nil
}

func haveToken(ctx context.Context) error {
	if from(ctx).sess.Token == "" {
		return errors.New("no session token was issued")
	}
	return nil
}

func logOut(ctx context.Context) error {
	w := from(ctx)
	if w.sess.Token == "" {
		return errors.New("not logged in")
	}
	w.auth.Logout(w.sess.Token)
	return nil
}

func minutesPass(ctx context.Context, n int) {
	w := from(ctx)
	w.now = w.now.Add(time.Duration(n) * time.Minute)
}

func sessionIs(ctx context.Context, want string) error {
	w := from(ctx)
	_, ok := w.auth.Validate(w.sess.Token)
	if got := map[bool]string{true: "valid", false: "invalid"}[ok]; got != want {
		return fmt.Errorf("expected session to be %s but it is %s", want, got)
	}
	return nil
}
