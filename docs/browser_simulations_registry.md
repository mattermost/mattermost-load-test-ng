# Browser Simulations Registry

This document lists all available browser simulations that can be run using the browser controller. Programmatically it is defined in the [registry.ts file](../browser/src/registry.ts). This gives an overview of the flow of the simulation and the actions that are performed.

## Simulations list

#### Post and Scroll scenario

**Id:** `postAndScroll`

**Description:** A simulation that mimics typical user behavior by posting messages and scrolling through channel history.

**Flow:**
1. Opens the Mattermost server URL in the browser
2. Handles the preference checkbox on the landing page if present
3. Logs in using the provided credentials
4. Selects the first team if team selection is required in the team selection page
5. Continuously loops through the following actions:
   - Navigates to the `town-square` channel
   - Posts a message in the `town-square` channel
   - Scrolls through the channel history (40 scroll actions with 400-500ms delays)
   - Navigates to the `off-topic` channel
   - Posts a message in the `off-topic` channel
   - Scrolls through the channel history (40 scroll actions with 400-500ms delays)
