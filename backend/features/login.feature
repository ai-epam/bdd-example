Feature: Login
  As an operator of the authorization console
  I want to sign in with my credentials
  So that only authenticated people can authorize transactions and reset data

  Background:
    Given the system is seeded with default data

  Scenario: Log in with valid credentials
    When I log in as "alice" with password "wonderland1"
    Then the login is "successful"
    And I receive a session token

  Scenario: Reject a wrong password
    When I log in as "alice" with password "not-my-password"
    Then the login is "rejected"
    And the login failure reason is "INVALID_CREDENTIALS"

  Scenario: Reject an unknown user without revealing that the user does not exist
    When I log in as "mallory" with password "whatever123"
    Then the login is "rejected"
    And the login failure reason is "INVALID_CREDENTIALS"

  Scenario: Reject a login with an empty password
    When I log in as "alice" with password ""
    Then the login is "rejected"
    And the login failure reason is "MISSING_CREDENTIALS"

  Scenario: Lock the account after three failed attempts
    Given I fail to log in as "bob" 3 times
    When I log in as "bob" with password "builder123"
    Then the login is "rejected"
    And the login failure reason is "ACCOUNT_LOCKED"

  Scenario: A locked account unlocks after five minutes
    Given I fail to log in as "bob" 3 times
    And 5 minutes pass
    When I log in as "bob" with password "builder123"
    Then the login is "successful"

  Scenario: A successful login resets the failed attempt counter
    Given I fail to log in as "bob" 2 times
    And I am logged in as "bob" with password "builder123"
    And I fail to log in as "bob" 2 times
    When I log in as "bob" with password "builder123"
    Then the login is "successful"

  Scenario: Logging out ends the session
    Given I am logged in as "alice" with password "wonderland1"
    And my session is "valid"
    When I log out
    Then my session is "invalid"

  Scenario: A session expires after thirty minutes
    Given I am logged in as "alice" with password "wonderland1"
    When 31 minutes pass
    Then my session is "invalid"
