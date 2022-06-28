[[_TOC_]]

## Synopsis

The `gcp_vault_secret` is an Ansible module for retrieving secret data from GCP Secret Manager.

If you want to compile it for local testing, you can do it:
```bash
mkdir -p $HOME/.ansible/plugins/modules
go get -d -v ./...
go build -x -o $HOME/.ansible/plugins/modules/gcp_vault_secret .
```

**Important**: module's code is executed on remote hosts! Hence you must compile the binary with `GOOS` and `GOARCH` of the target host (not your laptop or whatever you're running ansible from).

## Parameters

| parameter | required | default | choices | comments |
|-----------|----------|---------|---------|----------|
| name      | yes      | None    |         | Name of the secret in GCP Secret Manager |
| creds_file | no      | /tmp/.ansible/gcp_vault_secret_creds.json    |    file path or "system"      | Path to Google API credentials file on remote filesystem or use "system" if the default service account has permissions | 
| project_id | no      | taken from creds_file, specify if "system" is used |         | Name of the GCP Project where Secret Manager resides |
| private_google_api_endpoint      | no      | no    |    yes/no     | Make all requests to Google API via privately routed endpoint (private.googleapis.com:443) | 


## Example

Example of usage in a playbook (for this example you will need to export `GOOGLE_APPLICATION_CREDENTIALS` variable with local path to Google API creds file on your local host):
```yaml
- name: Test module gcp_vault_secret
  tasks:
    - name: Copy GCP credentials file to remote host
      copy:
        src: "{{ lookup('env', 'GOOGLE_APPLICATION_CREDENTIALS') }}"
        dest: /tmp/.ansible/gcp_vault_secret_creds.json
        mode: '0600'
        remote_src: no

    - name: Retrieve private key
      gcp_vault_secret:
        name: "my-secret-name"
        private_google_api_endpoint: no
      register: ssl_private_key

    - name: Save secret key to disk
      copy:
        content: "{{ ssl_private_key.data }}\n"
        dest: /etc/pki/tls/private/ssl.key
```

## System Example

This is an example where the default service account on a system has read access to secret manager.
```yaml
--- 
- name: Test System Service Account
  gcp_vault_secret:
    name: "my-secret-name"
    private_google_api_endpoint: yes
    creds_file: "system"
    project_id: "gcp-project-name"
  register: my_secret_value
```
## Return Values

| name | description | type | sample |
|------|-------------|------|--------|
| data | Plain-text secret data retrieved from Secret Manager | string | Bla!bla@MyPassWORd |
