;;; goasciinema.el --- Emacs interface for goasciinema -*- lexical-binding: t -*-

;; Author: Generated
;; Version: 1.0.0
;; Package-Requires: ((emacs "27.1"))
;; Keywords: tools, terminal, asciinema
;; URL: https://github.com/ober/goasciinema

;;; Commentary:

;; This package provides an Emacs interface for goasciinema, a terminal
;; session recorder.  It includes:
;;
;; - Interactive search with fancy result display
;; - Session listing and management
;; - Recording controls
;; - Playback integration
;;
;; Configuration:
;;   Database path is read from ~/.goasciinema config file:
;;     database = ~/console-logs/asciinema_logs.db
;;
;; Main commands:
;;   `goasciinema-search'       - Search terminal sessions
;;   `goasciinema-list'         - List all sessions
;;   `goasciinema-stats'        - Show database statistics
;;   `goasciinema-process'      - Process recordings (current dir)
;;   `goasciinema-process-path' - Process recordings at specific path
;;   `goasciinema-record'       - Start a new recording
;;   `goasciinema-play'         - Play a recording

;;; Code:

(require 'org)
(require 'ansi-color)

;;; Customization

(defgroup goasciinema nil
  "Interface for goasciinema terminal recorder."
  :group 'tools
  :prefix "goasciinema-")

(defcustom goasciinema-executable "goasciinema"
  "Path to the goasciinema executable."
  :type 'string
  :group 'goasciinema)

;; Database path is read from ~/.goasciinema config file by the Go tool.
;; No need to configure it in Emacs.

(defcustom goasciinema-search-context 5
  "Number of context lines to show before and after search matches."
  :type 'integer
  :group 'goasciinema)

(defcustom goasciinema-search-limit 50
  "Maximum number of search results to return."
  :type 'integer
  :group 'goasciinema)

(defcustom goasciinema-play-speed 1.0
  "Default playback speed multiplier."
  :type 'float
  :group 'goasciinema)

(defcustom goasciinema-play-idle-time-limit nil
  "Limit idle time during playback (seconds).
If nil, no limit is applied."
  :type '(choice (const :tag "No limit" nil)
                 (float :tag "Seconds"))
  :group 'goasciinema)

(defcustom goasciinema-record-command nil
  "Command to record.  If nil, uses $SHELL."
  :type '(choice (const :tag "Use $SHELL" nil)
                 (string :tag "Command"))
  :group 'goasciinema)

(defcustom goasciinema-record-idle-time-limit nil
  "Limit recorded idle time (seconds).
If nil, no limit is applied."
  :type '(choice (const :tag "No limit" nil)
                 (float :tag "Seconds"))
  :group 'goasciinema)

(defcustom goasciinema-record-title nil
  "Default title for recordings."
  :type '(choice (const :tag "No title" nil)
                 (string :tag "Title"))
  :group 'goasciinema)

(defcustom goasciinema-process-force nil
  "If non-nil, force reprocessing of already processed files."
  :type 'boolean
  :group 'goasciinema)

;;; Faces

(defface goasciinema-match-face
  '((t :inherit highlight :weight bold))
  "Face for matched text in search results."
  :group 'goasciinema)

(defface goasciinema-filename-face
  '((t :inherit font-lock-function-name-face :weight bold))
  "Face for filenames in results."
  :group 'goasciinema)

(defface goasciinema-date-face
  '((t :inherit font-lock-comment-face))
  "Face for dates in results."
  :group 'goasciinema)

(defface goasciinema-line-number-face
  '((t :inherit line-number))
  "Face for line numbers."
  :group 'goasciinema)

(defface goasciinema-context-face
  '((t :inherit default))
  "Face for context lines."
  :group 'goasciinema)

;;; Internal variables

(defvar goasciinema-search-history nil
  "History of search terms.")

(defvar goasciinema-current-search-term nil
  "Current search term for highlighting.")

;;; Utility functions

(defun goasciinema--build-args (command &rest args)
  "Build argument list for COMMAND with ARGS.
Database path is configured via ~/.goasciinema, not passed here."
  (cons command (delq nil args)))

(defun goasciinema--run-command (command &rest args)
  "Run goasciinema COMMAND with ARGS and return output as string."
  (with-temp-buffer
    (let ((exit-code (apply #'call-process
                            goasciinema-executable
                            nil t nil
                            (apply #'goasciinema--build-args command args))))
      (if (zerop exit-code)
          (buffer-string)
        (error "goasciinema %s failed: %s" command (buffer-string))))))

(defun goasciinema--run-command-async (command buffer-name callback &rest args)
  "Run goasciinema COMMAND asynchronously.
Output goes to BUFFER-NAME, CALLBACK called on completion with exit code."
  (let ((buffer (get-buffer-create buffer-name)))
    (with-current-buffer buffer
      (let ((inhibit-read-only t))
        (erase-buffer)))
    (make-process
     :name (format "goasciinema-%s" command)
     :buffer buffer
     :command (cons goasciinema-executable
                    (apply #'goasciinema--build-args command args))
     :sentinel (lambda (proc _event)
                 (when (memq (process-status proc) '(exit signal))
                   (funcall callback (process-exit-status proc)))))))

;;; Search functionality

(defun goasciinema-search (term)
  "Search for TERM in asciinema recordings."
  (interactive
   (list (read-string "Search term: " nil 'goasciinema-search-history)))
  (setq goasciinema-current-search-term term)
  (let ((output (goasciinema--run-command
                 "search" term
                 "-c" (number-to-string goasciinema-search-context)
                 "-n" (number-to-string goasciinema-search-limit))))
    (goasciinema--display-search-results output term)))

(defun goasciinema--display-search-results (output term)
  "Display search OUTPUT for TERM in a dedicated buffer."
  (let ((buffer (get-buffer-create "*goasciinema-search*")))
    (with-current-buffer buffer
      (let ((inhibit-read-only t))
        (erase-buffer)
        (insert output)
        (goto-char (point-min))
        ;; Handle carriage returns in output
        ;; First: normalize Windows line endings (\r\n -> \n)
        (while (search-forward "\r\n" nil t)
          (replace-match "\n" nil t))
        (goto-char (point-min))
        ;; Second: handle mid-line carriage returns (terminal overwrites)
        ;; Keep only text after the last \r on each line
        (while (re-search-forward "^.*\r" nil t)
          (replace-match "" nil t))
        (goto-char (point-min))
        ;; Third: fix terminal soft line wraps
        ;; Only join when an alphanumeric char is followed by padding spaces,
        ;; newline, and alphanumeric or hyphen+alphanumeric (for CLI args like -search)
        ;; This avoids merging lines ending with punctuation or followed by >>> etc
        (while (re-search-forward "\\([[:alnum:]]\\)  +\n\\(-?[[:alnum:]]\\)" nil t)
          (replace-match "\\1 \\2" nil nil))
        (goto-char (point-min))
        ;; Highlight matches
        (goasciinema--highlight-matches term)
        (org-mode)
        (goasciinema-search-mode)
        (setq-local goasciinema-current-search-term term)
        (goto-char (point-min))))
    (switch-to-buffer buffer)))

(defun goasciinema--highlight-matches (term)
  "Highlight occurrences of TERM in current buffer."
  (save-excursion
    (goto-char (point-min))
    (let ((case-fold-search t))
      (while (re-search-forward (regexp-quote term) nil t)
        (let ((overlay (make-overlay (match-beginning 0) (match-end 0))))
          (overlay-put overlay 'face 'goasciinema-match-face))))))

(defvar goasciinema-search-mode-map
  (let ((map (make-sparse-keymap)))
    (define-key map (kbd "n") #'goasciinema-search-next-match)
    (define-key map (kbd "p") #'goasciinema-search-prev-match)
    (define-key map (kbd "s") #'goasciinema-search)
    (define-key map (kbd "q") #'quit-window)
    (define-key map (kbd "RET") #'goasciinema-search-open-session)
    (define-key map (kbd "g") #'goasciinema-search-refresh)
    map)
  "Keymap for `goasciinema-search-mode'.")

(define-minor-mode goasciinema-search-mode
  "Minor mode for goasciinema search results.

\\{goasciinema-search-mode-map}"
  :lighter " GoAsc-Search"
  :keymap goasciinema-search-mode-map
  (setq buffer-read-only t))

(defun goasciinema-search-next-match ()
  "Jump to next search match."
  (interactive)
  (when goasciinema-current-search-term
    (let ((case-fold-search t))
      (if (re-search-forward (regexp-quote goasciinema-current-search-term) nil t)
          (goto-char (match-beginning 0))
        (message "No more matches")))))

(defun goasciinema-search-prev-match ()
  "Jump to previous search match."
  (interactive)
  (when goasciinema-current-search-term
    (let ((case-fold-search t))
      (if (re-search-backward (regexp-quote goasciinema-current-search-term) nil t)
          (goto-char (match-beginning 0))
        (message "No more matches")))))

(defun goasciinema-search-refresh ()
  "Refresh the current search."
  (interactive)
  (when goasciinema-current-search-term
    (goasciinema-search goasciinema-current-search-term)))

(defun goasciinema-search-open-session ()
  "Open the session file at point (if in org heading)."
  (interactive)
  (save-excursion
    (org-back-to-heading t)
    (let ((heading (org-get-heading t t t t)))
      (when (string-match "Match [0-9]+: \\(.+\\)" heading)
        (let ((filename (match-string 1 heading)))
          (message "Session: %s" filename))))))

;;; List sessions

(defun goasciinema-list ()
  "List all processed asciinema sessions."
  (interactive)
  (let ((output (goasciinema--run-command "list")))
    (goasciinema--display-list output)))

(defun goasciinema--display-list (output)
  "Display session list OUTPUT in a dedicated buffer."
  (let ((buffer (get-buffer-create "*goasciinema-list*")))
    (with-current-buffer buffer
      (let ((inhibit-read-only t))
        (erase-buffer)
        (insert output)
        (goto-char (point-min))
        (goasciinema-list-mode)))
    (switch-to-buffer buffer)))

(defvar goasciinema-list-mode-map
  (let ((map (make-sparse-keymap)))
    (define-key map (kbd "g") #'goasciinema-list)
    (define-key map (kbd "q") #'quit-window)
    (define-key map (kbd "s") #'goasciinema-search)
    map)
  "Keymap for `goasciinema-list-mode'.")

(define-minor-mode goasciinema-list-mode
  "Minor mode for goasciinema session list.

\\{goasciinema-list-mode-map}"
  :lighter " GoAsc-List"
  :keymap goasciinema-list-mode-map
  (setq buffer-read-only t))

;;; Statistics

(defun goasciinema-stats ()
  "Display database statistics."
  (interactive)
  (let ((output (goasciinema--run-command "stats")))
    (message "%s" (string-trim output))))

;;; Process files

(defun goasciinema-process ()
  "Process asciinema recordings into the database.
Uses the current directory.  Database path is configured via ~/.goasciinema."
  (interactive)
  (let ((args (if goasciinema-process-force '("-f") nil)))
    (apply #'goasciinema--run-command-async
           "process"
           "*goasciinema-process*"
           (lambda (exit-code)
             (if (zerop exit-code)
                 (message "Processing complete")
               (message "Processing failed with exit code %d" exit-code)))
           args)))

(defun goasciinema-process-path (path)
  "Process asciinema recordings at PATH into the database.
Database path is configured via ~/.goasciinema."
  (interactive "fPath to process: ")
  (let ((args (if goasciinema-process-force (list "-f" path) (list path))))
    (apply #'goasciinema--run-command-async
           "process"
           "*goasciinema-process*"
           (lambda (exit-code)
             (if (zerop exit-code)
                 (message "Processing complete")
               (message "Processing failed with exit code %d" exit-code)))
           args)))

;;; Recording

(defun goasciinema-record (filename)
  "Start recording a terminal session to FILENAME."
  (interactive "FOutput file: ")
  (let* ((args (list "rec" filename))
         (args (if goasciinema-record-command
                   (append args (list "-c" goasciinema-record-command))
                 args))
         (args (if goasciinema-record-title
                   (append args (list "-t" goasciinema-record-title))
                 args))
         (args (if goasciinema-record-idle-time-limit
                   (append args (list "-i" (number-to-string goasciinema-record-idle-time-limit)))
                 args)))
    ;; Recording needs a real terminal, open in term
    (let ((buffer (apply #'make-term "goasciinema-record"
                         goasciinema-executable
                         nil
                         args)))
      (switch-to-buffer buffer)
      (term-mode)
      (term-char-mode)
      (message "Recording started. Exit shell to stop recording."))))

;;; Playback

(defun goasciinema-play (filename)
  "Play back the recording at FILENAME."
  (interactive "fRecording file: ")
  (let* ((args (list "play" filename))
         (args (if (not (= goasciinema-play-speed 1.0))
                   (append args (list "-s" (number-to-string goasciinema-play-speed)))
                 args))
         (args (if goasciinema-play-idle-time-limit
                   (append args (list "-i" (number-to-string goasciinema-play-idle-time-limit)))
                 args)))
    ;; Playback needs a real terminal
    (let ((buffer (apply #'make-term "goasciinema-play"
                         goasciinema-executable
                         nil
                         args)))
      (switch-to-buffer buffer)
      (term-mode)
      (term-char-mode))))

;;; Cat (view full output)

(defun goasciinema-cat (filename)
  "Display the full output of recording FILENAME."
  (interactive "fRecording file: ")
  (let ((output (goasciinema--run-command "cat" filename))
        (buffer (get-buffer-create "*goasciinema-output*")))
    (with-current-buffer buffer
      (let ((inhibit-read-only t))
        (erase-buffer)
        (insert output)
        (ansi-color-apply-on-region (point-min) (point-max))
        (goto-char (point-min))
        (special-mode)))
    (switch-to-buffer buffer)))

;;; Auth and Upload

(defun goasciinema-auth ()
  "Authenticate with asciinema.org."
  (interactive)
  (let ((output (goasciinema--run-command "auth")))
    (message "%s" (string-trim output))))

(defun goasciinema-upload (filename)
  "Upload recording FILENAME to asciinema.org."
  (interactive "fRecording to upload: ")
  (message "Uploading %s..." filename)
  (goasciinema--run-command-async
   "upload"
   "*goasciinema-upload*"
   (lambda (exit-code)
     (if (zerop exit-code)
         (with-current-buffer "*goasciinema-upload*"
           (message "Upload complete: %s" (string-trim (buffer-string))))
       (message "Upload failed")))
   filename))

;;; Transient menu (if available)

(when (require 'transient nil t)
  (transient-define-prefix goasciinema-menu ()
    "Goasciinema commands."
    ["Search & Browse"
     ("s" "Search" goasciinema-search)
     ("l" "List sessions" goasciinema-list)
     ("S" "Statistics" goasciinema-stats)]
    ["Recording"
     ("r" "Record" goasciinema-record)
     ("p" "Play" goasciinema-play)
     ("c" "Cat (view output)" goasciinema-cat)]
    ["Database"
     ("P" "Process (current dir)" goasciinema-process)
     ("D" "Process path..." goasciinema-process-path)]
    ["Online"
     ("a" "Authenticate" goasciinema-auth)
     ("u" "Upload" goasciinema-upload)]))

;;; Global keymap (optional)

(defvar goasciinema-command-map
  (let ((map (make-sparse-keymap)))
    (define-key map (kbd "s") #'goasciinema-search)
    (define-key map (kbd "l") #'goasciinema-list)
    (define-key map (kbd "S") #'goasciinema-stats)
    (define-key map (kbd "r") #'goasciinema-record)
    (define-key map (kbd "p") #'goasciinema-play)
    (define-key map (kbd "c") #'goasciinema-cat)
    (define-key map (kbd "P") #'goasciinema-process)
    (define-key map (kbd "D") #'goasciinema-process-path)
    (define-key map (kbd "a") #'goasciinema-auth)
    (define-key map (kbd "u") #'goasciinema-upload)
    map)
  "Keymap for goasciinema commands.
Bind this to a prefix key, e.g.:
  (global-set-key (kbd \"C-c a\") goasciinema-command-map)")

(provide 'goasciinema)
;;; goasciinema.el ends here
