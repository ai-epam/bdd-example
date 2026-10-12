Feature: Card transaction authorization
  As a payments risk manager
  I want every card transaction to be checked against our authorization policy
  So that only compliant spend is approved

  Background:
    Given the system is seeded with default data

  Scenario: Approve a valid coffee purchase within all limits
    Given card "CARD-001" is active with balance "500.00" USD
    And merchant "MERCH-COFFEE" has MCC "5812" in country "US"
    When I authorize "12.50" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    Then the transaction is "approved"
    And the account balance of "ACC-001" becomes "487.50" USD

  Scenario: Decline when the card is frozen
    Given card "CARD-002" is frozen
    And merchant "MERCH-COFFEE" has MCC "5812" in country "US"
    When I authorize "12.50" USD on card "CARD-002" at merchant "MERCH-COFFEE"
    Then the transaction is "declined"
    And the decline reason is "CARD_FROZEN"

  Scenario: Decline when the amount exceeds the per-transaction limit
    Given card "CARD-004" has a per-transaction limit of "20.00" USD
    And merchant "MERCH-GROCERY" has MCC "5411" in country "US"
    When I authorize "25.00" USD on card "CARD-004" at merchant "MERCH-GROCERY"
    Then the transaction is "declined"
    And the decline reason is "AMOUNT_EXCEEDS_PER_TXN_LIMIT"

  Scenario: Decline when the account has insufficient funds
    Given merchant "MERCH-GROCERY" has MCC "5411" in country "US"
    When I authorize "75.00" USD on card "CARD-003" at merchant "MERCH-GROCERY"
    Then the transaction is "declined"
    And the decline reason is "INSUFFICIENT_FUNDS"
