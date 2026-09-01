# Install Mechazone on a shop laptop

This is the picture-by-picture guide for a shop owner or technician. You do **not** need to be a programmer.

After install, daily use is: **double-click Mechazone**, sign in, plug the OpenPort, scan.

---

## What you need

| Item | Notes |
| --- | --- |
| The laptop that will sit in the bay | Ubuntu / Debian Linux, or Windows 10/11. 64-bit. |
| This Mechazone folder | A USB stick, a zip download, or a copy from another shop. Keep it somewhere permanent (Documents is fine). **Do not delete the folder** after install — the app lives inside it. |
| Internet, the first time | The installer downloads a few pieces. After that, the bay works offline; VIN decode needs the radio when you want it. |
| Your computer password | Linux will ask once, so the installer can add the database and USB permission. |
| OpenPort 2.0 Rev E + USB cable | For a real car. You can install and practise **without** the cable (Mock ECU). |

You do **not** need a second laptop, a tablet, or an ELM327. The product is this software on the OpenPort you already own.

---

## Linux (Ubuntu / Debian) — one command

1. Copy the Mechazone folder onto the laptop. Example: `Documents/mechazone`.
2. Open that folder in the file manager.
3. Right-click empty space → **Open in Terminal**.
4. Type this and press Enter:

```bash
./install.sh
```

5. If the laptop asks for your password, type it (nothing will show as you type) and press Enter.
6. Wait. The first run can take several minutes. When it says **Done**, you are finished.

The installer will:

- Add any missing laptop pieces (Python, PostgreSQL, USB support)
- Download Go and Node if this laptop does not already have recent ones
- Create a local database named `mechazone`
- Build the shop screen and the ledger
- Put a **Mechazone** icon on the Desktop and in the app menu
- Try to allow the OpenPort USB cable (log out and back in before a real car)

If a line starts with `ERROR:`, stop and use [Troubleshooting](#troubleshooting).

---

## Windows

1. Install these four programs if they are not already on the laptop (use the default options; tick **Add to PATH** for Python):
   - [Python 3](https://www.python.org/downloads/)
   - [Node.js LTS](https://nodejs.org/)
   - [Go](https://go.dev/dl/) (1.24 or newer)
   - [PostgreSQL](https://www.postgresql.org/download/windows/) — remember the postgres password you set
2. In **pgAdmin** or SQL Shell, create a database named `mechazone` if the installer cannot create it.
3. Open the Mechazone folder.
4. Right-click `install.ps1` → **Run with PowerShell**.  
   If Windows blocks it: open PowerShell in that folder and run:

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\install.ps1
```

5. When it says **Done**, double-click **Mechazone** on the Desktop.

For a real car on Windows, do **not** install drivers from tactrix.com (that updater can wipe a clone, even on a PC that is online for other work). Put a frozen, clone-matched PassThru DLL in the Mechazone folder and set `J2534_LIB=` to that file. Block `tactrix.com` if you want a belt-and-braces. Details: `docs/integrations.md`.

---

## First login (every shop)

1. Start Mechazone (Desktop icon, or `scripts/start-mechazone.sh` on Linux).
2. A browser window should open to `http://127.0.0.1:8080`. The public page is a landing page — request access, or click **Issued a login?**
3. Sign in with the seeded super admin:

| | |
| --- | --- |
| Email | `admin@mechazone.local` |
| Password | `change-me-now` |

4. **Change that password** as soon as you can (edit `SUPERADMIN_PASSWORD` in the `.env` file in the Mechazone folder, then stop and start the app). Do not leave the default password on a shop floor.

There is **no self-signup**. The super admin creates shops and issues logins.

---

## After you are in: add shops and technicians

Still signed in as the super admin:

1. Open **Admin**.
2. Add the shop (name is enough). Freelancers can exist without a shop.
3. Add each technician: email, password, shop (or leave shop empty for a freelancer).
4. Hand that email and password to the technician. They sign in on the same laptop (or another installed laptop) and use the bay.

The ledger tags every scan with the signed-in technician (and shop, if they have one). The bay does not invent those IDs.

---

## Every working day

1. Double-click **Mechazone**.
2. Sign in as a technician (not the admin) to scan and log jobs.
3. To practise without a car: choose **Mock ECU**, then Connect → Identify → Scan.
4. For a real car:
   - Ignition on
   - OpenPort USB in the laptop (SD card **out** of the OpenPort)
   - Cable on the OBD port
   - Choose **OpenPort 2.0 Rev E**
   - Connect → Identify (VIN from the module) → read history → Scan → log the job → closeout
5. Customer name, phone, and plate stay on this laptop. Only VIN and mechanical facts go to the ledger.

To stop the app on Linux:

```bash
./scripts/stop-mechazone.sh
```

On Windows, sign out of the browser tab and, if needed, end `mechazone-server` in Task Manager — or run `scripts\stop-mechazone.ps1`.

---

## What the installer does *not* require

- A second interface (no ELM327, no extra dongle)
- Vite, three terminals, or “run the client”
- Creating your own account on the internet
- Rebuilding VIN decoders or fault-code books (those are already wired; see `docs/integrations.md`)

---

## Troubleshooting

**The terminal says “permission denied” for `./install.sh`**  
Run: `chmod +x install.sh scripts/start-mechazone.sh scripts/stop-mechazone.sh` and try again.

**It asks for a password and then fails on packages**  
You need sudo on this laptop. Ask whoever set up the computer to run `./install.sh` once.

**“Could not create the mechazone database”**  
PostgreSQL is installed but this user cannot create databases. Ask for help with:

```bash
sudo -u postgres createdb -O "$USER" mechazone
```

Then run `./install.sh` again.

**The browser does not open, or the page is blank**  
On Linux, in the Mechazone folder:

```bash
./scripts/stop-mechazone.sh
./scripts/start-mechazone.sh
```

If it still fails, open `var/server.log` in the Mechazone folder and look at the last lines. The usual cause is the database not running:

```bash
sudo service postgresql start
```

**Login works but Connect does nothing**  
The worker did not start. Check `var/worker.log`. Re-run `./install.sh`. Mock ECU should work without any cable.

**OpenPort is plugged in but the bay cannot see the car**  
- SD card removed from the OpenPort  
- Ignition on  
- Log out and back in once after install (USB group)  
- On Linux, `J2534_LIB` in `.env` should point at `third_party/j2534/j2534/j2534.so` if that file exists  
- More hardware detail: `docs/integrations.md`

**We moved the folder after install**  
Run `./install.sh` again from the new place so the Desktop icon and settings point at the new path.

**Developers (not for the shop floor)**  
`make backend`, `make worker`, and `make client` still work. Daily shop use is the installer + Desktop icon, which serves the built UI from port 8080. Third-party keys and OpenPort compile notes stay in `docs/integrations.md`.
