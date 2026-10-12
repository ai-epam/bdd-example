Feature: Refunds
  As an account holder
  I want a refund to reverse the original purchase
  So that I get my money back when a merchant cancels

  Background:
    Given the system is seeded with default data

  # DEMO CENTERPIECE: this scenario fails on purpose. The refund behavior is not
  # implemented yet (see the TODO in backend/internal/authz/rules.go).
  @wip
  Scenario: Refund reverses a prior debit
    Given card "CARD-001" is active with balance "500.00" USD
    And merchant "MERCH-COFFEE" has MCC "5812" in country "US"
    When I authorize "12.50" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    And I refund the last transaction
    Then the account balance of "ACC-001" becomes "500.00" USD
