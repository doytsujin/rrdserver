# Paths ..........................
PREFIX     = /usr/local
CONFIGDIR  = /etc
SYSTEMDDIR = /usr/lib/system
INITDIR    = /etc/init.d
BINDIR     = $(PREFIX)/sbin

CONFIG_FILE     = $(CONFIGDIR)/rrdserver.conf
BINARY          = $(BINDIR)/rrdserver
SYSTEMD_SERVICE = $(SYSTEMDDIR)/rrdserver.service
INITD_FILE      = $(INITDIR)/rrdservice

# Build variables ................
MAKEFILE_DIR ?= $(realpath $(dir $(lastword $(MAKEFILE_LIST))))
BUILD_DIR = $(MAKEFILE_DIR)/.build

#SRC_FILES    = $(wildcard *.go */*.go */*/*.go)
#VERSION     := $(shell awk -F'"' '/RRDSERVER_VERSION/ {print($$2)}' main.go)

GOPATH ?= $(BUILD_DIR)/.TMP_GOROOT
PROJECT = github.com/rrdserver/rrdserver
PROJECT_PATH = $(GOPATH)/src/$(PROJECT)
#***********************************************************
define title
    @echo -e "\033[0;32m*$1\033[0m"
endef

all: build systemd initd

$(BUILD_DIR):
	@$(call title, "Create .build directory")
	@mkdir -p $(BUILD_DIR)/{src,bin,pkg}
	
$(GOPATH): $(BUILD_DIR)
	@$(call title, "Create root directory")
	@mkdir -p $(GOPATH)/{src,bin,pkg}

	
$(PROJECT_PATH): $(GOPATH)
	@$(call title, "Create directory for rrdserver project")
	@mkdir -p $(GOPATH)/src/$(dir $(PROJECT))
	@ln -s $(MAKEFILE_DIR) $(GOPATH)/src/$(PROJECT)

	
build: $(GOPATH) $(PROJECT_PATH)
	@$(call title, "Build binary")
	@GOPATH=$(GOPATH) go get $(PROJECT) 
	@GOPATH=$(GOPATH) go build -o $(BUILD_DIR)/rrdserver $(PROJECT)
	@install -D -m 644 $(MAKEFILE_DIR)/rrdserver.conf $(BUILD_DIR)/rrdserver.conf
	
systemd: $(BUILD_DIR)	
	@$(call title, "Create systemd service file")
	@sed -e "s|/usr/local/sbin/rrdserver|$(BINARY)|" \
	     -e "s|/etc/rrdserver.conf|$(CONFIG_FILE)|" \
	     $(MAKEFILE_DIR)/misc/systemd/rrdserver.service > $(BUILD_DIR)/rrdserver.service
	
initd: $(BUILD_DIR)
	@$(call title, "Create init.d file")
	@sed -e "s|/usr/local/sbin/rrdserver|$(BINARY)|" \
	     -e "s|/etc/rrdserver.conf|$(CONFIG_FILE)|" \
	     -e "s|/etc/init.d|$(INITDIR)|" \
	     $(MAKEFILE_DIR)/misc/init.d/rrdserver > $(BUILD_DIR)/rrdserver.init_script

install:
	install -D -m 755 $(BUILD_DIR)/${BINARY}          $(DESTDIR)/$(BINDIR)
	install -D -m 644 $(BUILD_DIR)/$(CONFIG_FILE )    $(DESTDIR)/$(CONFIG_FILE)
	install -D -m 644 $(BUILD_DIR)/${SYSTEMD_SERVICE} $(DESTDIR)/$(SYSTEMD_SERVICE)
	install -D -m 755 $(BUILD_DIR)/$(INITD_FILE)      $(DESTDIR)/$(INITD_FILE)
	
clean:	
	@rm -rf $(BUILD_DIR)

	