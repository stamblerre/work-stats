package reviews

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/build/maintner"
)

type ReviewData struct {
	// CLs authored
	authored map[*maintner.GerritCL]struct{}

	// CLs reviewed
	reviewed map[*maintner.GerritCL]struct{}
}

// Get some statistics on issues opened, closed, and commented on.
func Data(gerrit *maintner.Gerrit, email string, start time.Time) (*ReviewData, error) {
	stats := &ReviewData{
		authored: make(map[*maintner.GerritCL]struct{}),
		reviewed: make(map[*maintner.GerritCL]struct{}),
	}
	var ownerID int
	gerrit.ForeachProjectUnsorted(func(project *maintner.GerritProject) error {
		project.ForeachCLUnsorted(func(cl *maintner.GerritCL) error {
			if cl.Status != "merged" {
				return nil
			}
			// If the user created the CL.
			if cl.Owner() != nil && cl.Owner().Email() == email {
				ownerID = cl.OwnerID()
				if cl.Created.After(start) {
					stats.authored[cl] = struct{}{}
				}
				return nil
			}
			// If the user reviewed the CL.
			for _, msg := range cl.Messages {
				// If the user's email is actually tracker.
				// Not sure why this happens for some people, but not others.
				if msg.Author != nil && msg.Author.Email() == email {
					if msg.Date.After(start) {
						stats.reviewed[cl] = struct{}{}
						return nil
					}
				}
				if strings.HasPrefix(msg.Author.Name(), "Gerrit User") {
					split := strings.Split(msg.Author.Name(), " ")
					if len(split) == 3 {
						id, err := strconv.ParseInt(split[2], 10, 64)
						if err != nil {
							log.Fatal(err)
						}
						if int(id) == ownerID {
							if msg.Date.After(start) {
								stats.reviewed[cl] = struct{}{}
								return nil
							}
						}
					}
				}
			}
			return nil
		})
		return nil
	})
	return stats, nil
}

func Write(outputDir string, stats *ReviewData) ([]string, error) {
	// Write out authored CLs.
	filename1, err := writeCLs(outputDir, "authored.csv", stats.authored)
	if err != nil {
		return nil, err
	}
	filename2, err := writeCLs(outputDir, "reviewed.csv", stats.reviewed)
	if err != nil {
		return nil, err
	}
	return []string{filename1, filename2}, nil
}

func writeCLs(dir, filename string, stats map[*maintner.GerritCL]struct{}) (string, error) {
	// Write out authored CLs.
	path := filepath.Join(dir, filename)
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	var total int
	var sorted []*maintner.GerritCL
	for cl := range stats {
		sorted = append(sorted, cl)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Created.Before(sorted[j].Created)
	})
	writer.Write([]string{"CL", "Description"})
	for _, cl := range sorted {
		writer.Write([]string{
			// TODO: Technically should insert the -review into cl.Project.Server().
			fmt.Sprintf("go-review.googlesource.com/c/%s/+/%v", cl.Project.Project(), cl.Number),
			cl.Subject(),
		})
		total++
	}
	writer.Write([]string{"Total", fmt.Sprintf("%v", total)})

	return path, nil
}
