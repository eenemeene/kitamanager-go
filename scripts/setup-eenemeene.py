#!/usr/bin/env python3
"""Setup script for the 'Eene Meene' organization."""

import argparse
import sys
from pathlib import Path

import requests

parser = argparse.ArgumentParser(description="Setup the 'Eene Meene' organization.")
parser.add_argument("base_url", nargs="?", default="http://localhost:8080")
parser.add_argument("email", nargs="?", default="admin@example.com")
parser.add_argument("password", nargs="?", default="supersecret")
parser.add_argument("--with-funding-bills", type=Path, metavar="DIR",
                    help="Directory with ISBJ Abrechnungen (.xlsx) to upload recursively")
args = parser.parse_args()

BASE_URL = args.base_url
EMAIL = args.email
PASSWORD = args.password
PROJECT_DIR = Path(__file__).resolve().parent.parent

API = f"{BASE_URL}/api/v1"
session = requests.Session()


def api(method: str, path: str, **kwargs) -> requests.Response:
    resp = session.request(method, f"{API}{path}", **kwargs)
    if not resp.ok:
        print(f"FAILED: {method} {path} -> {resp.status_code} {resp.text}", flush=True)
        sys.exit(1)
    return resp


# --- 1. Login ---------------------------------------------------------

print(f"Logging in as {EMAIL} ...", flush=True)
api("POST", "/login", json={"email": EMAIL, "password": PASSWORD})

access_token = session.cookies.get("access_token")
csrf_token = session.cookies.get("csrf_token")
if not access_token:
    print("FAILED: could not extract access_token from cookies", flush=True)
    sys.exit(1)
if not csrf_token:
    print("FAILED: could not extract csrf_token from cookies", flush=True)
    sys.exit(1)

session.headers.update({
    "Authorization": f"Bearer {access_token}",
    "X-CSRF-Token": csrf_token,
})
print("  OK", flush=True)

# --- 2. Create organization -------------------------------------------

print("Creating organization 'Eene Meene' ...", flush=True)
org_resp = api("POST", "/organizations", json={
    "name": "Eene Meene",
    "active": True,
    "state": "berlin",
    "default_section_name": "Default",
})
org_id = org_resp.json()["id"]
print(f"  OK (org_id={org_id})", flush=True)

# --- 3. Import government funding rates --------------------------------

print("Importing government funding rates from configs/government-fundings/berlin.yaml ...", flush=True)
with open(PROJECT_DIR / "configs/government-fundings/berlin.yaml", "rb") as f:
    resp = session.post(f"{API}/government-funding-rates/import", params={"state": "berlin"}, files={"file": f})
    if resp.status_code == 409:
        print("  OK (already exists)", flush=True)
    elif not resp.ok:
        print(f"FAILED: POST /government-funding-rates/import -> {resp.status_code} {resp.text}", flush=True)
        sys.exit(1)
    else:
        print("  OK", flush=True)

# --- 4. Import pay plans ----------------------------------------------

print("Importing pay plans from configs/pay-plans/tv-eene-meene.yaml ...", flush=True)
with open(PROJECT_DIR / "configs/pay-plans/tv-eene-meene.yaml", "rb") as f:
    api("POST", f"/organizations/{org_id}/pay-plans/import", files={"file": f})
print("  OK", flush=True)

# --- 5. Import employees ----------------------------------------------

print("Importing employees from configs/employees/eene-meene.yaml ...", flush=True)
with open(PROJECT_DIR / "configs/employees/eene-meene.yaml", "rb") as f:
    api("POST", f"/organizations/{org_id}/employees/import", files={"file": f})
print("  OK", flush=True)

# --- 6. Import children -----------------------------------------------

print("Importing children from configs/children/eene-meene.yaml ...", flush=True)
with open(PROJECT_DIR / "configs/children/eene-meene.yaml", "rb") as f:
    api("POST", f"/organizations/{org_id}/children/import", files={"file": f})
print("  OK", flush=True)

# --- 7. Update section age ranges -------------------------------------

print("Fetching sections ...", flush=True)
sections = api("GET", f"/organizations/{org_id}/sections", params={"limit": 100}).json()

section_by_name = {s["name"]: s["id"] for s in sections["data"]}

SECTION_AGES = {
    "Nest":          {"min_age_months": 0, "max_age_months": 36},
    "Nestfluechter": {"min_age_months": 36, "max_age_months": 48},
    "Gross":         {"min_age_months": 48},
}

print("Updating section age ranges ...", flush=True)
for name, body in SECTION_AGES.items():
    sid = section_by_name.get(name)
    if not sid:
        print(f"FAILED: section '{name}' not found", flush=True)
        sys.exit(1)
    api("PUT", f"/organizations/{org_id}/sections/{sid}", json=body)
    print(f"  {name}: {body}", flush=True)

# --- 8. Upload Abrechnungen (optional) --------------------------------

if args.with_funding_bills:
    bills_dir = args.with_funding_bills
    if not bills_dir.is_dir():
        print(f"FAILED: '{bills_dir}' is not a directory", flush=True)
        sys.exit(1)

    xlsx_files = sorted(bills_dir.rglob("*.xlsx"), reverse=True)
    if not xlsx_files:
        print(f"No .xlsx files found in '{bills_dir}'", flush=True)
    else:
        print(f"Uploading {len(xlsx_files)} Abrechnung(en) from '{bills_dir}' ...", flush=True)
        for xlsx in xlsx_files:
            rel = xlsx.relative_to(bills_dir)
            print(f"  Uploading {rel} ...", flush=True, end="")
            with open(xlsx, "rb") as f:
                api("POST", f"/organizations/{org_id}/government-funding-bills",
                    files={"file": (xlsx.name, f,
                                    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")})
            print(" OK", flush=True)

print(f"\nDone! Organization 'Eene Meene' (id={org_id}) is ready.", flush=True)
