describe("Open Endpoints", () => {
  it("website", () => {
    cy.visit("/");
  });

  it("status", () => {
    cy.request(Cypress.env("statusUrl") + "/status")
      .its("body")
      .should("have.property", "status", "online");
  });
});
