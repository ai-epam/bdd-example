Feature: Merchant and country rules
  As a compliance officer
  I want to block high-risk merchant categories and unsupported countries
  So that cards are not used where policy forbids

  Background:
    Given the system is seeded with default data

  Scenario: Decline a gambling merchant on a card that blocks MCC 7995
    Given merchant "MERCH-GAMBLING" has MCC "7995" in country "US"
    When I authorize "10.00" USD on card "CARD-001" at merchant "MERCH-GAMBLING"
    Then the transaction is "declined"
    And the decline reason is "MCC_BLOCKED"

  Scenario: Approve a gambling merchant on a card that does not block it
    Given card "CARD-900" on account "ACC-001" does not block any MCC
    And merchant "MERCH-GAMBLING" has MCC "7995" in country "US"
    When I authorize "10.00" USD on card "CARD-900" at merchant "MERCH-GAMBLING"
    Then the transaction is "approved"

  Scenario: Decline a Korean merchant on a card restricted to the US
    Given merchant "MERCH-FOREIGN" has MCC "5411" in country "KR"
    When I authorize "10.00" USD on card "CARD-004" at merchant "MERCH-FOREIGN"
    Then the transaction is "declined"
    And the decline reason is "COUNTRY_BLOCKED"

  Scenario: Approve a US merchant on a card restricted to the US
    Given merchant "MERCH-GROCERY" has MCC "5411" in country "US"
    When I authorize "10.00" USD on card "CARD-004" at merchant "MERCH-GROCERY"
    Then the transaction is "approved"
