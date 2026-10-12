Feature: Transaction velocity limits
  As a fraud analyst
  I want to cap how many transactions a card can make in a short window
  So that stolen card numbers cannot be drained quickly

  Background:
    Given the system is seeded with default data

  Scenario: Approve when under the velocity limit
    Given merchant "MERCH-COFFEE" has MCC "5812" in country "US"
    And 4 approved transactions on card "CARD-001" in the last 30 seconds
    When I authorize "5.00" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    Then the transaction is "approved"

  Scenario: Decline the transaction that exceeds the velocity limit inside the window
    Given merchant "MERCH-COFFEE" has MCC "5812" in country "US"
    And 5 approved transactions on card "CARD-001" in the last 30 seconds
    When I authorize "5.00" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    Then the transaction is "declined"
    And the decline reason is "VELOCITY_LIMIT_EXCEEDED"

  Scenario: The velocity window slides so older transactions stop counting
    Given merchant "MERCH-COFFEE" has MCC "5812" in country "US"
    And 5 approved transactions on card "CARD-001" 90 seconds ago
    When I authorize "5.00" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    Then the transaction is "approved"

  Scenario: Declined transactions do not count toward velocity
    Given merchant "MERCH-COFFEE" has MCC "5812" in country "US"
    When I authorize "250.00" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    And I authorize "250.00" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    And I authorize "250.00" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    And I authorize "250.00" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    And I authorize "250.00" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    And I authorize "10.00" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    Then the transaction is "approved"
