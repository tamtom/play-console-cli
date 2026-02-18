# Wall of Apps

Apps and teams using **gplay CLI** to manage their Google Play releases.

## Add Your App

Submit a PR that adds your app to `docs/wall-of-apps.json`:

### Entry format

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | App display name |
| `package` | Yes | Android package name |
| `developer` | No | Developer or organization |
| `url` | No | Website or Play Store link |
| `description` | No | Brief app description |

### Example entry

```json
{
  "name": "My App",
  "package": "com.example.myapp",
  "developer": "Example Inc.",
  "url": "https://play.google.com/store/apps/details?id=com.example.myapp",
  "description": "A great app that does things"
}
```
