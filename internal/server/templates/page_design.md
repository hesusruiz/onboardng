# Context of the design 

There are two HTML pages, index.hbs and detail.hbs.
The first one (registration_list) displays a list of registrations, and the second one (registration_detail) displays the details of a specific registration.

The source of the data to be displayed are the tables described in the SQL queries below, which are used to create the tables in the SQLite database.

The objective is to design the html pages to display the data from the SQLite database in a user-friendly and organized manner. For the moment, we focus ONLY in the details page, which is the most complex.

It is not the objective to implement the SQLite queries to fetch the data from the database, but to use mock data to test the layout. The focus at this moment is in the design and mock-up of the page. The current HTML page is just a barebones layout where each region will be filled with relevant data fetched from the SQLite database.

We focus first on the details page.

## SQLite tables used

```sql
	CREATE TABLE IF NOT EXISTS registrations (
		registration_id TEXT UNIQUE,
		email TEXT UNIQUE,
		first_name TEXT,
		last_name TEXT,
		company_name TEXT,
		country TEXT,
		vat_id TEXT UNIQUE,
		street_address TEXT,
		city TEXT,
		postal_code TEXT,
		role TEXT,
		notified BOOLEAN,
		issued BOOLEAN,
		tmf_registered BOOLEAN,
		created_at DATETIME,
		updated_at DATETIME,
		lr_first_name TEXT,
		lr_last_name TEXT,
		lr_email TEXT,
		lr_country TEXT,
		lr_id_card TEXT,
		lear_first_name TEXT,
		lear_last_name TEXT,
		lear_email TEXT,
		lear_country TEXT,
		lear_address TEXT,
		lear_id_card TEXT,
		lear_mobile_number TEXT,
		lear_completed BOOLEAN,
		files_uploaded BOOLEAN,
		approved INTEGER
	);
	CREATE TABLE IF NOT EXISTS registration_log (
		registration_id TEXT,
		email TEXT,
		vat_id TEXT,
		type TEXT,
		message TEXT,
		created_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS registration_files (
		file_id TEXT UNIQUE,
		registration_id TEXT,
		vat_id TEXT,
		name TEXT,
		mime_type TEXT,
		size INTEGER,
		status TEXT,
		content BLOB,
		created_at DATETIME,
		updated_at DATETIME
	);
```

## Page detail.hbs

The page has a layout defined using the CSS Grid, and the basic structure of the page is defined using HTML/CSS.
The page template with the overall structure and layout of the page is as follows:

```html
<!DOCTYPE html>
<html lang="en">

<head>
    <title>AdminReg</title>
    <link rel="stylesheet" href="assets/w35.css" />
    <link rel="stylesheet" href="assets/dome.css" />
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/4.7.0/css/font-awesome.min.css">
    <link
        href="https://fonts.googleapis.com/css2?family=Blinker:wght@100;200;300;400;600;700;800;900&family=Caladea:ital,wght@0,400;0,700;1,400;1,700&display=swap"
        rel="stylesheet">
    <link rel="stylesheet"
        href="https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@20..48,100..700,0..1,-50..200&icon_names=check_circle,counter_1,counter_2,counter_3" />


    <style>
        body {
            background-color: #FBE9E7;
        }

        .company-profile {
            grid-area: company-profile;
        }

        .person-registering {
            grid-area: person-registering;
        }

        .legal-representative {
            grid-area: legal-representative;
        }

        .account-admin {
            grid-area: account-admin;
        }

        .uploaded-documents {
            grid-area: uploaded-documents;
        }

        .process-status {
            grid-area: process-status;
        }

        .log {
            grid-area: log;
        }

        .gridwrapper {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            grid-template-rows: repeat(5, 1fr);
            grid-auto-columns: 1fr;
            gap: 16px 16px;
            grid-auto-flow: row;
            grid-template-areas:
                "company-profile company-profile process-status"
                "person-registering person-registering process-status"
                "legal-representative legal-representative process-status"
                "account-admin account-admin process-status"
                "uploaded-documents uploaded-documents ."
                "log log log";
        }

        .gridwrapper>div {
            border: 1px solid grey;
            background-color: white;
            padding: 16px;
            border-radius: 8px;
            box-shadow: 3px 3px 3px 3px rgba(0, 0, 0, 0.2);
            height: auto;
            width: auto;
        }
    </style>


</head>

<body>
    <div class="w3-container">

        <div class="gridwrapper">
            <div class="company-profile">
                <h3>Company Profile</h3>
            </div>
            <div class="person-registering">
                <h3>Person Registering</h3>
            </div>
            <div class="legal-representative">
                <h3>Legal Representative</h3>
            </div>
            <div class="account-admin">
                <h3>Account Admin</h3>
            </div>
            <div class="uploaded-documents">
                <h3>Uploaded Documents</h3>
            </div>
            <div class="process-status">
                <h3>Process Status</h3>
            </div>
            <div class="log">
                <h3>Log</h3>
            </div>
        </div>
    </div>
</body>

</html>
```

Which is a barebones layout where each region will be filled with relevant data fetched from the SQLite database. As mentioned above, we first will use mock data to design the layout, then we will replace the mock data with actual data from the SQLite database.

Regarding the design, I want to emphasize the following points:

- The page is divided into 7 regions, where we use standard CSS grid definitions for the structure. We will NOT use any CSS framework for the definition of the layout.
- The regions are displayed in a grid layout, with 3 columns and 5 rows, where some regions use more than one column or row, using the CSS grid-column and grid-row properties.
- In the template, the regions do not have content. It is the objective of this task to define the content, using the information described below.

### Section Company Profile

Uses table registrations, displaying the following information:

Legal name: company_name
VAT number: vat_id
Country: country
Street address: street_address
Postal code: postal_code
City: city

### Section Person Registering

Uses table registrations, displaying the following information:

Name: first_name + last_name
Email: email

### Section Legal Representative

Uses table registrations, displaying the following information:

Name: lr_first_name + lr_last_name
Email: lr_email
Nationality: lr_country
ID Card: lr_id_card

### Section Account Admin

Uses table registrations, displaying the following information:

Name: lear_first_name + lear_last_name
Email: lear_email
Nationality: lear_country
Address: lear_address
ID Card: lear_id_card
Mobile Number: lear_mobile_number

### Section Uploaded Documents

This is a list of zero or more documents uploaded by the user during the registration process. For each document, we have the following information, using the table registration_files:

Name: name
Mime type: mime_type
Size: size
Status: status
Created at: created_at
Updated at: updated_at

### Section Process Status

Displays an overview of the status of the registration process.
The registration process consists on several stages:

Stage 1: The user has completed the initial registration form, with the information about himself/herself and the company.
After this stage, the registrations table has the information needed for the sections "Company Profile" and "Person Registering".
The system tries to do the following:
- Send a notification email to the user, telling him that the registration process was OK. If the system can send the email, the field `notified` is set to true. Otherwise, the field `notified` is set to false.
- The system issues a credential to the user, using the API of the Issuer application. If the system can issue the credential, the field `issued` is set to true. Otherwise, the field `issued` is set to false.
- The system creates an Organization object in the TM Forum server, using the API of the Server application. If the system can create the organization object, the field `tmf_registered` is set to true. Otherwise, the field `tmf_registered` is set to false.

If any errors happened during these operations, they will be logged in the `registration_log` table, and they will show up in the `log` section of the page.

The `process_status` section shows the current status of the registration process. The status is determined by the values of the fields `notified`, `issued`, and `tmf_registered` in the `registrations` table.

If one or more of the flags is false, the `process_status` section will display a button that could be used by the admin user to restart/retry the individial action that the flag represents and that failed at the time of registration.

Stage 2: The user has completed the second registration form, with the information about the legal representative and the account admin. It is not possible that the user only fills one of this sections only.
When this info has been stored in the `registrations` table, the flag `lr_completed` is set to true when this stage is completed.

Stage 3: During this stage, the user can upload one or more documents, which will be shown in the `uploaded-documents` section. The flag `uploaded` is set to true when this stage is completed.

Stage 4: The admin user has verified all the data, including the uploaded documents, and has formally approved the registration. The process_status section shows the field `approved` which has three possible values: "Pending Approval", "Approved", and "Rejected". The process_status section shows three buttons that the admin user can click and that changes the value of the `approved`field in th edatabase.


### Section Log

This section shows the records of the table `registration_log` in chronological order.
The fields to show in each log entry are:

Timestamp: created_at. It has to be displayed in human readable format, and with seconds precission.
Type: type. This can have the values Error, Warning, Info
Message: message

If the type is Error, the log entry will be displayed in red.
If the type is Warning, the log entry will be displayed in orange.
If the type is Info, the log entry will be displayed in green.


## Page index.hbs

The index.hbs page is a list of the registrations in the table registrations, and each list item must show the important information about each registration.
The page uses the same visual style as the detail.hbs page, and each column in each list item must show:



### Column with type of company

Seller or Buyer, stressed with an icon representing a seller or a buyer.

### Column with dates

`created_at` and `updated_at`

### Column for the Company Profile

Legal name: company_name
VAT number: vat_id
Country: country

### Column for Person Registering

Uses table registrations, displaying the following information:

Name: first_name + last_name
Email: email

### Column with visual tags for each of the Stage 1 flags

The flags are `notified`, `issued`, `tmf_registered`

### Column Legal Representative

Uses table registrations, displaying the following information:

Name: lr_first_name + lr_last_name
Email: lr_email

### Column Account Admin

Name: lear_first_name + lear_last_name
Email: lear_email

### Column with visual tags for each of the Stage 2 flags

The flags are `lr_completed`, `uploaded`.

### Column with global status of the registration

Visual tag for flag `approved`, with the thrre possible states
Must include a button or link to go to the details page corresponding to that registration.

