# `provenance` CLI tool

This tool fetches a JSON file containing GCP machine info/provenance from a public GCS bucket based on a PPID(Platform Provisioning Id).

## Building the Tool

To create the executable, run the following command from the repository root:

```console
go build ./tools/provenance
```

This will create a `provenance` executable in your current directory.

Alternatively, you can run the tool directly without building an executable:

```console
go run ./tools/provenance [options...]
```

## Usage
The PPID can be determined in three ways:

### 1. Automatically from the local platform
Requires running inside a supported Confidential VM with TDX and root privileges (`sudo`):
```bash
sudo ./provenance
```

### 2. Extracted from a TDX Quote file
Path to a raw binary TDX quote file to extract PPID from. The quote must contain a PCK certificate chain from which the PPID is extracted.

```bash
./provenance -quote /path/to/quote.bin
```

### 3. Provided directly via flag
```bash
./provenance -ppid <PPID_STRING>
```

## Optional Flags

- `-bucket`: The public GCS bucket name to fetch the provenance document from.
- `-out`: Path to output file to write provenance data to. Defaults to stdout.
- `-verbose`: Enable verbose output.
## PPID Resolution Order

If neither `-ppid` nor `-quote` is provided, the tool will attempt to fetch a TDX quote from the local system. This usually requires root privileges (`sudo`).
