//go:build !symphony_publication_faults || (!darwin && !linux)

package shvpublicationstate

// Fault barriers are absent from production builds.
func publicationBarrier(string) {}
