//go:build !darwin && !linux

package credential

func openJournalFile(*Journal, string, bool) (journalFile, error) { return nil, errJournal }
