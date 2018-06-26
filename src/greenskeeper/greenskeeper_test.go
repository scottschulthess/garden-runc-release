package greenskeeper

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Greenskeeper", func() {
	Describe("#CheckExistingGdnProcess", func() {
		var (
			pidFile *os.File
		)

		BeforeEach(func() {
			var err error
			pidFile, err = ioutil.TempFile("", "pidfile")
			Expect(err).NotTo(HaveOccurred())
			pidFile.Close()
		})

		AfterEach(func() {
			os.Remove(pidFile.Name())
			Expect(pidFile.Name()).NotTo(BeAnExistingFile())
		})

		It("removes stale pidfiles", func() {
			Expect(CheckExistingGdnProcess(pidFile.Name())).To(Succeed())
			Expect(pidFile.Name()).NotTo(BeAnExistingFile())
		})

		Context("when removing the pidfile fails", func() {
			It("returns an error", func() {
				failingRemove := func(string) error { return errors.New("I failed") }
				Expect(checkExistingGdnProcess(pidFile.Name(), failingRemove)).NotTo(Succeed())
			})
		})

		Context("when the pidfile doesn't exist", func() {
			It("does not return an error", func() {
				Expect(CheckExistingGdnProcess("/does/not/exist")).To(Succeed())
			})
		})

		Context("when the PIDFILE refers to a running process", func() {
			var (
				sleepSession *gexec.Session
				sleepPid     string
			)

			BeforeEach(func() {
				sleepSession = gexecStart(exec.Command("sleep", "60"), GinkgoWriter, GinkgoWriter)
				sleepPid = strconv.Itoa(sleepSession.Command.Process.Pid)

				Expect(ioutil.WriteFile(pidFile.Name(), []byte(sleepPid), os.ModePerm)).To(Succeed())
			})

			AfterEach(func() {
				sleepSession.Kill().Wait()
			})

			It("returns an error", func() {
				err := CheckExistingGdnProcess(pidFile.Name())
				Expect(err).To(MatchError(fmt.Sprintf("garden is already running (pid: %s)", sleepPid)))
			})
		})
	})

	Describe("#CopyFiles", func() {
		var (
			sourceDir      string
			destDir        string
			origKittenFile string
			newKittenFile  string
			files          []File

			copyErr error
		)

		BeforeEach(func() {
			var err error
			sourceDir, err = ioutil.TempDir("", "source")
			Expect(err).NotTo(HaveOccurred())
			destDir, err = ioutil.TempDir("", "dest")
			Expect(err).NotTo(HaveOccurred())

			origKittenFile = filepath.Join(sourceDir, "kitten")
			newKittenFile = filepath.Join(destDir, "kitten")

			err = ioutil.WriteFile(origKittenFile, []byte("i am very fluffy"), 0640)
			Expect(err).NotTo(HaveOccurred())

			files = []File{File{
				Source: origKittenFile,
				Dest:   newKittenFile,
				UID:    1001,
				GID:    1002,
			}}
		})

		JustBeforeEach(func() {
			copyErr = CopyFiles(files)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(sourceDir)).To(Succeed())
			Expect(os.RemoveAll(destDir)).To(Succeed())
		})

		It("does not fail", func() {
			Expect(copyErr).NotTo(HaveOccurred())
		})

		Context("when it copies the file", func() {
			var kittenFileInfo os.FileInfo

			JustBeforeEach(func() {
				kittenFileInfo = stat(newKittenFile)
			})

			It("'places' (writes) it in the correct destination", func() {
				Expect(newKittenFile).To(BeAnExistingFile())
			})

			It("maintains the correct permissions", func() {
				Expect(kittenFileInfo.Mode().Perm()).To(Equal(os.FileMode(0640)))
			})

			It("sets the correct user ownership", func() {
				Expect(kittenFileInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(1001)))
			})

			It("sets the correct group ownership", func() {
				Expect(kittenFileInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(1002)))
			})

			It("has the correct file contents", func() {
				origKittenFileContents, err := ioutil.ReadFile(origKittenFile)
				Expect(err).NotTo(HaveOccurred())

				newKittenFileContents, err := ioutil.ReadFile(newKittenFile)
				Expect(err).NotTo(HaveOccurred())

				Expect(string(newKittenFileContents)).To(Equal(string(origKittenFileContents)))
			})

			Context("when the file already exists at the destination", func() {
				BeforeEach(func() {
					Expect(CopyFiles(files)).To(Succeed())
				})

				It("does not fail", func() {
					Expect(copyErr).NotTo(HaveOccurred())
				})

				Context("and that file is a running executable", func() {
					BeforeEach(func() {
						var err error
						sleepyCmd := exec.Command(sleepyBin)
						_, err = gexec.Start(sleepyCmd, GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())

						dst, err := filepath.Abs(sleepyBin)
						Expect(err).NotTo(HaveOccurred())
						files = []File{File{
							Source: sleepyBin2,
							Dest:   dst,
							UID:    1001,
							GID:    1002,
							Remove: true,
						}}
					})

					It("does not fail", func() {
						Expect(copyErr).NotTo(HaveOccurred())
					})
				})
			})
		})
	})

	Describe("#SetupDirectories", func() {
		var (
			dir         string
			pikachuDir  string
			setupErr    error
			directories []Directory
		)

		BeforeEach(func() {
			var err error
			dir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())
			pikachuDir = path.Join(dir, "pikachu")
			directories = []Directory{NewDirectoryBuilder(pikachuDir).Mode(0644).
				UID(1001).GID(1002).Build()}
		})

		JustBeforeEach(func() {
			setupErr = CreateDirectories(directories...)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(dir)).To(Succeed())
		})

		It("does not fail", func() {
			Expect(setupErr).NotTo(HaveOccurred())
		})

		It("sets up a directory", func() {
			By("creating the directory")
			Expect(pikachuDir).To(BeAnExistingFile())

			By("setting the correct permissions")
			fileInfo := stat(pikachuDir)
			Expect(fileInfo.Mode().Perm()).To(Equal(os.FileMode(0644)))

			By("setting the correct user ownership")
			Expect(fileInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(1001)))

			By("setting the correct group ownership")
			Expect(fileInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(1002)))
		})

		Context("when creating a directory fails", func() {
			BeforeEach(func() {
				directories[0].mkdirAll = func(string, os.FileMode) error { return errors.New("I failed") }
			})

			It("returns an error", func() {
				Expect(setupErr).To(HaveOccurred())
			})
		})

		Context("when chowning the directory fails", func() {
			BeforeEach(func() {
				directories[0].chown = func(string, int, int) error { return errors.New("I failed") }
			})

			It("returns an error", func() {
				Expect(setupErr).To(HaveOccurred())
			})
		})

		Context("when user and group are not specified", func() {
			BeforeEach(func() {
				directories = []Directory{NewDirectoryBuilder(pikachuDir).Build()}
				directories[0].chown = func(string, int, int) error {
					return errors.New("I should not have been called")
				}
			})

			It("should not chown", func() {
				Expect(setupErr).NotTo(HaveOccurred())
			})
		})

		Context("when group only is specfied", func() {
			BeforeEach(func() {
				directories = []Directory{NewDirectoryBuilder(pikachuDir).GID(1001).Build()}
			})

			It("should chown using current user ID", func() {
				Expect(stat(pikachuDir).Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(0)))
				Expect(stat(pikachuDir).Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(1001)))
			})
		})

		Context("when chmodding the directory fails", func() {
			BeforeEach(func() {
				directories[0].chmod = func(string, os.FileMode) error { return errors.New("I failed") }
			})

			It("returns an error", func() {
				Expect(setupErr).To(HaveOccurred())
			})
		})

		Context("when Mode isn't specified", func() {
			BeforeEach(func() {
				directories[0].Mode = nil
			})
			It("defaults to the permissions when system umask is applied", func() {
				Expect(stat(pikachuDir).Mode().Perm()).To(Equal(applyUmask(defaultDirectoryMode)))
			})
		})

		Context("when setting up two directories", func() {
			BeforeEach(func() {
				directories = append(directories, NewDirectoryBuilder(path.Join(dir, "charmander")).UID(1001).GID(1002).Build())
			})

			It("sets up both", func() {
				Expect(setupErr).NotTo(HaveOccurred())

				By("creating both directories")
				Expect(pikachuDir).To(BeAnExistingFile())
				Expect(path.Join(dir, "charmander")).To(BeAnExistingFile())
			})
		})

		Context("when the process umask is 0007", func() {
			var oldUmask int

			BeforeEach(func() {
				oldUmask = syscall.Umask(0007)
				directories[0].Mode = newFileMode(0777)
			})

			AfterEach(func() {
				syscall.Umask(oldUmask)
			})

			It("sets expected permissions", func() {
				fileInfo := stat(pikachuDir)
				Expect(fileInfo.Mode().Perm()).To(Equal(os.FileMode(0777)))
			})
		})
	})
})

func gexecStart(cmd *exec.Cmd, stdout, stderr io.Writer) *gexec.Session {
	session, err := gexec.Start(cmd, stdout, stderr)
	Expect(err).NotTo(HaveOccurred())
	return session
}

func applyUmask(mode os.FileMode) os.FileMode {
	oldUmask := syscall.Umask(0)
	syscall.Umask(oldUmask)
	return os.FileMode(int(mode) &^ oldUmask)
}

func stat(path string) os.FileInfo {
	fileInfo, err := os.Stat(path)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return fileInfo
}
