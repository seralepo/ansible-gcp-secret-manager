`gcp_vault_secret` is an Ansible module for retrieving secret data from GCP Secret Manager.

If you want to compile it for local testing, you can do it:
```bash
mkdir -p $HOME/.ansible/plugins/modules
go get -d -v ./...
go build -x -o $HOME/.ansible/plugins/modules/gcp_vault_secret .
```

Example of usage in a playbook (for this example you will need to export `GOOGLE_APPLICATION_CREDENTIALS` variable with local path to Google API creds file on your local host):
```yaml
---
- name: Test module gcp_vault_secret
  tasks:
    - name: Copy GCP credentials file to remote host
      copy:
        src: "{{ lookup('env', 'GOOGLE_APPLICATION_CREDENTIALS') }}"
        dest: /tmp/gcp_vault_secret_creds.json
        mode: '0600'
        remote_src: no

    - name: Retrieve private key
      gcp_vault_secret:
        name: "my-secret-name"
        creds_file: /tmp/gcp_vault_secret_creds.json
        project_id: "my-gcp-project-id"
        private_google_api_endpoint: no
      register: ssl_private_key

    - name: Save secret key disk
      copy:
        content: "{{ ssl_private_key.data }}\n"
        dest: /etc/pki/tls/private/ssl.key
```

With setting `private_google_api_endpoint` to `yes`, all API requests to GCP will be routed via `private.googleapis.com:443` instead of real endpoints.