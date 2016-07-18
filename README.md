#CU CeotGrade (CU 出Grade)

This application will send you an email notification whenever your grades has been updated.

##Configuration
    const (
    	sid      = "1155012345"
    	pw       = "my password"
    	email    = "my gmail@gmail.com"
    	emailpw  = "my gmail pw"
    	smtpHost = "smtp.gmail.com"
	    smtpAddr = "smtp.gmail.com:587"
    )
Change the sid, pw, email and emailpw in the source file. Secure Gmail users who have employed
2-step verification will have to obtain an "App specific password" for emailpw.
You may change smtpHost and smtpAddr if you want to use mail services other than Gmail.

##Usage
Deploy it to OpenShift to run 24 hours/day for free, and then use Uptime Robot to prevent it from sleeping.
You may run it on your own computer too.

##Running example
[http://ceotgrade-asdsteven.rhcloud.com/](http://ceotgrade-asdsteven.rhcloud.com/)
