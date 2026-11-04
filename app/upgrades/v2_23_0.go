package upgrades

// Upgrade2_23_0 activates the NibiruBFT block lane rules. It runs only the
// default module migrations and changes no stores. ProcessProposal enforces
// the lane rules from the height after this upgrade's done record, so the
// binary that runs this upgrade is the binary that starts enforcing them.
// See app.LaneRulesUpgradeName.
var Upgrade2_23_0 = NewVanillaUpgrade("v2.23.0")
