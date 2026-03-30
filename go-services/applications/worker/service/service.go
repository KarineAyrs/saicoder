package service

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/KarineAyrs/safe-ai-coder/applications/worker/domain"
	"github.com/KarineAyrs/safe-ai-coder/applications/worker/interfaces"
	"github.com/KarineAyrs/safe-ai-coder/pkg/config"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"go.opencensus.io/trace"
)

type Service struct {
	logger	log.Logger
	coderCfg config.Coder

	scheduler interfaces.Scheduler
	sem	   chan struct{}
}

func NewService(scheduler interfaces.Scheduler, coderCfg config.Coder) *Service {
	return &Service{
		logger:	log.NewNopLogger(),
		coderCfg: coderCfg,
		scheduler: scheduler,
		sem:	   make(chan struct{}, 1),
	}
}

func (s *Service) WithLogger(l log.Logger) *Service {
	s.logger = l
	return s
}

func (s *Service) Submit(ctx context.Context, t domain.Task) error {
	op := "Service.Submit"
	ctx, span := trace.StartSpan(ctx, op)
	defer span.End()

	err := s.submit(t)
	if err != nil {
		return fmt.Errorf("submit: %w", err)
	}

	return nil
}

func (s *Service) submit(t domain.Task) error {
	select {
	case s.sem <- struct{}{}:
		go func() {
			defer func() { <-s.sem }()

			res := s.processTask(t)

			ctx := context.Background()  // TODO: WithTimeout
			err := s.scheduler.UpdateTask(ctx, res)
			if err != nil {
				level.Error(s.logger).Log("msg", "update task failed", "err", err)
			}
		}()
		return nil
	default:
		return errors.New("worker is already busy")
	}
}

func (s *Service) processTask(t domain.Task) domain.Result {
	errRes := domain.Result{
		Status: domain.Failed,
		ID:	 t.ID,
	}

	zipPath := s.getTaskResultPath(t.ID)
	exists, err := checkFileExists(zipPath)
	if err != nil {
		level.Error(s.logger).Log("msg", "failed to check if file exists", "err", err)
		return errRes
	}

	if !exists {
		cmd := exec.Command("rm", "-rf", s.coderCfg.CodeDir)
		if err := cmd.Run(); err != nil {
			level.Error(s.logger).Log("msg", "failed to cleanup workdir", "err", err)
			return errRes
		}

		cmd = exec.Command("mkdir", "-p", s.coderCfg.CodeDir)
		if err := cmd.Run(); err != nil {
			level.Error(s.logger).Log("msg", "failed to mkdir workdir", "err", err)
			return errRes
		}

		path := filepath.Join(s.coderCfg.CodeDir, s.coderCfg.StatementFilename)
		err = os.WriteFile(path, []byte(t.Statement), 0644)
		if err != nil {
			level.Error(s.logger).Log("msg", "failed to write statement", "err", err)
			return errRes
		}

		cmdArgs := []string{
			"-p", "safe-ai-coder",
			"-f", "/docker-compose.yml",
			"--profile", "tools",
			"run", "--rm", "coder",
			"-w", s.coderCfg.CodeDir,
			"-s", s.coderCfg.StatementFilename,
			"-m", s.coderCfg.ModelName,
			"--ollama-base-url", s.coderCfg.OllamaBaseURL,
		}
		if s.coderCfg.WithChecker {
			cmdArgs = append(cmdArgs, "--enable-checker")
		} else {
			cmdArgs = append(cmdArgs, "--disable-checker")
		}

		cmd = exec.Command("/usr/local/bin/docker-compose", cmdArgs...)
		cmd.Stderr = os.Stderr

		err := os.MkdirAll(s.coderCfg.StatsDir, 0755)
		if err != nil {
			level.Error(s.logger).Log("msg", "failed to mkdir stats dir", "err", err)
			return errRes
		}

		outfile, err := os.Create(s.getTaskStatsPath(t.ID))
		if err != nil {
			level.Error(s.logger).Log("msg", "failed to create stats file", "err", err)
			return errRes
		}
		defer outfile.Close()
		cmd.Stdout = outfile

		if err = cmd.Run(); err != nil {
			level.Error(s.logger).Log("msg", "failed to run coder", "err", err)
			return errRes
		}

		err = zipDirectory(s.coderCfg.CodeDir, zipPath)
		if err != nil {
			level.Error(s.logger).Log("msg", "failed to zip directory", "err", err)
			return errRes
		}
	}

	b64, err := fileToBase64(zipPath)
	if err != nil {
		level.Error(s.logger).Log("msg", "failed to base64 zip archive", "err", err)
		return errRes
	}

	return domain.Result{
		Status: domain.Done,
		Base64: b64,
		ID:	 t.ID,
	}
}

func (s *Service) getTaskResultPath(id string) string {
	return filepath.Join(s.coderCfg.CacheDir, fmt.Sprintf("%s.zip", id))
}

func (s *Service) getTaskStatsPath(id string) string {
	return filepath.Join(s.coderCfg.StatsDir, fmt.Sprintf("%s.json", id))
}

func checkFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("file stat: %w", err)
	}
	return true, nil
}

func zipDirectory(srcDir, zipPath string) error {
	err := os.MkdirAll(filepath.Dir(zipPath), 0755)
	if err != nil {
		return fmt.Errorf("mkdir all: %w", err)
	}

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("create zip: %w", err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	return filepath.WalkDir(srcDir, func(path string, dir os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dir.Name() == "." || dir.Name() == ".." {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return fmt.Errorf("filepath rel: %w", err)
		}

		if relPath == "." {
			return nil
		}

		if dir.IsDir() {
			info, err := dir.Info()
			if err != nil {
				return fmt.Errorf("dir info: %w", err)
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return fmt.Errorf("create header: %w", err)
			}

			header.Name = relPath + "/"
			header.Method = zip.Store
			_, err = w.CreateHeader(header)
			if err != nil {
				return fmt.Errorf("create directory in zip: %w", err)
			}

			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer f.Close()

		zipEntry, err := w.Create(relPath)
		if err != nil {
			return fmt.Errorf("create regular file in zip: %w", err)
		}

		_, err = io.Copy(zipEntry, f)
		if err != nil {
			return fmt.Errorf("copy file to zip: %w", err)
		}

		return nil
	})
}

func fileToBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file %q: %w", path, err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}
