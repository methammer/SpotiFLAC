package main

import bolt "go.etcd.io/bbolt"

// Container regroupe toutes les dépendances du service layer.
// Il est câblé dans main.go et passé explicitement à App et Server.
type Container struct {
	DB      *bolt.DB
	Jobs    *JobManager
	Auth    *AuthManager
	Watcher *Watcher
}
