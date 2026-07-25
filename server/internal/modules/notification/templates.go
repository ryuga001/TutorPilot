package notification

const (
	subjectEmailVerification = "Verify your Zenith AI email"
	subjectPasswordReset     = "Your Zenith AI password reset code"
)

const (
	// args: name, otp, link, link, minutes
	tmplEmailVerification = `<p>Hi %s,</p>
<p>Welcome to Zenith AI! Confirm your email with this code:</p>
<h2 style="letter-spacing:4px">%s</h2>
<p>Or open this link and enter the code:</p>
<p><a href="%s">%s</a></p>
<p>This code expires in %d minutes.</p>`

	// args: name, otp, minutes
	tmplPasswordReset = `<p>Hi %s,</p>
<p>Your password reset code is:</p>
<h2 style="letter-spacing:4px">%s</h2>
<p>It expires in %d minutes. If you didn't request this, ignore this email.</p>`
)
