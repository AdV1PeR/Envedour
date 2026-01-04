package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// checkDependencies проверяет наличие необходимых зависимостей
func checkDependencies() error {
	deps := []string{"yt-dlp", "aria2c", "ffmpeg"}
	missing := []string{}

	for _, dep := range deps {
		if _, err := exec.LookPath(dep); err != nil {
			missing = append(missing, dep)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("отсутствуют необходимые зависимости: %v\n\nУстановите их:\n  sudo apt-get install ffmpeg aria2 python3-pip\n  sudo pip3 install yt-dlp", missing)
	}

	return nil
}

// checkTmpfs проверяет доступность tmpfs директории
func checkTmpfs(tmpfsPath string) error {
	// Проверяем что директория существует или может быть создана
	if err := os.MkdirAll(tmpfsPath, 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию %s: %w", tmpfsPath, err)
	}

	// Проверяем что можем писать
	testFile := filepath.Join(tmpfsPath, ".test_write")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("нет доступа на запись в %s: %w\n\nПроверьте права доступа:\n  sudo chmod 1777 %s", tmpfsPath, err, tmpfsPath)
	}
	os.Remove(testFile)

	return nil
}

// ValidateEnvironment проверяет окружение перед запуском
func ValidateEnvironment(tmpfsPath string) error {
	if err := checkDependencies(); err != nil {
		return err
	}
	if err := checkTmpfs(tmpfsPath); err != nil {
		return err
	}

	return nil
}
