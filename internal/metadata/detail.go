package metadata

import "context"

// ItemDetail is the full view of a single metadata item: its indexed record
// (decorated with installed-app status) plus the rendered manifest content
// fetched on demand from the source repository. It backs the desktop UI's
// click-to-expand detail panel.
type ItemDetail struct {
	Item     Item   `json:"item"`
	Content  string `json:"content"`
	Manifest string `json:"manifest_path"`
}

// Detail returns the full detail view for one item by kind and install key.
// The indexed record comes from the local SQLite index (fast); the manifest
// content is fetched from the upstream repository on demand (slow, network).
// A fetch failure is non-fatal: the indexed metadata is still returned with an
// empty Content so the UI can render the summary without the manifest body.
func (svc *Service) Detail(ctx context.Context, kind, installKey string) (ItemDetail, error) {
	item, err := svc.store.GetItem(ctx, kind, installKey)
	if err != nil {
		return ItemDetail{}, err
	}
	item.InstalledApps = InstalledAppsFor(item)
	content, manifest := svc.fetchResourceManifest(item)
	return ItemDetail{Item: item, Content: content, Manifest: manifest}, nil
}
