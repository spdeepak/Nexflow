# Debug mode

To run the app in debug mode in GoLand
* Set `working directory` to `{FOLDER_WHERE_REPO_IS_LOCATES}/Nexflow/cmd/app`
* Add `Environment` variable `CGO_LDFLAGS=-framework UniformTypeIdentifiers`
* Add `GO Tool arguments` variable `-tags=dev` 