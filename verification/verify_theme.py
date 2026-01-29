from playwright.sync_api import sync_playwright

def verify_theme():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page()

        page.on("console", lambda msg: print(f"CONSOLE: {msg.text}"))

        page.goto("http://localhost:5173/")

        # Wait for the grid to appear
        page.wait_for_selector(".ag-root-wrapper", timeout=10000)

        page.screenshot(path="verification/fixed_state.png")
        browser.close()

if __name__ == "__main__":
    verify_theme()
