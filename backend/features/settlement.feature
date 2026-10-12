Feature: Settlement of authorizations
  As an account holder
  I want my balance to reflect exactly the purchases that were approved
  So that I am never charged for declined or duplicated transactions

  Background:
    Given the system is seeded with default data

  Scenario: A successful authorization debits the account exactly once
    Given card "CARD-001" is active with balance "500.00" USD
    And merchant "MERCH-COFFEE" has MCC "5812" in country "US"
    When I authorize "12.50" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    Then the transaction is "approved"
    And the account balance of "ACC-001" becomes "487.50" USD

  Scenario: A declined authorization leaves the balance untouched
    Given card "CARD-001" is active with balance "500.00" USD
    And merchant "MERCH-GAMBLING" has MCC "7995" in country "US"
    When I authorize "10.00" USD on card "CARD-001" at merchant "MERCH-GAMBLING"
    Then the transaction is "declined"
    And the account balance of "ACC-001" is unchanged
    And the account balance of "ACC-001" becomes "500.00" USD

  Scenario: Two approved transactions sum correctly
    Given card "CARD-001" is active with balance "500.00" USD
    And merchant "MERCH-COFFEE" has MCC "5812" in country "US"
    When I authorize "10.00" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    And I authorize "15.25" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    Then the transaction is "approved"
    And the account balance of "ACC-001" becomes "474.75" USD

  Scenario: The daily limit declines spend that would pass it
    Given card "CARD-001" is active with balance "500.00" USD
    And card "CARD-001" has a per-transaction limit of "200.00" USD
    And card "CARD-001" has a daily limit of "400.00" USD
    And merchant "MERCH-GROCERY" has MCC "5411" in country "US"
    When I authorize "150.00" USD on card "CARD-001" at merchant "MERCH-GROCERY"
    And I authorize "150.00" USD on card "CARD-001" at merchant "MERCH-GROCERY"
    And I authorize "150.00" USD on card "CARD-001" at merchant "MERCH-GROCERY"
    Then the transaction is "declined"
    And the decline reason is "DAILY_LIMIT_EXCEEDED"
    And the account balance of "ACC-001" is unchanged
    And the account balance of "ACC-001" becomes "200.00" USD
