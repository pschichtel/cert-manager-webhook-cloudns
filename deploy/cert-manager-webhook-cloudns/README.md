Usage
=====

1. Adjust `certManager.namespace` and `certManager.serviceAccountName` to align with your setup
2. Fill in the `clouDNS` section. `authId` and `authPassword` are secrets.

Secrets
=======

Secrets can be specified either as a string, which will be rendered into a secret by the helm chart,
or as a secretKeyRef.

Example as string:

```yaml
clouDNS:
  authPassword: "your-auth-password"
```

Example as secretKeyRef:

```yaml
clouDNS:
  authPassword:
    name: your-secret-name
    key: your-secret-key
```