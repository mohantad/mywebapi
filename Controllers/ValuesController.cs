using System;
using System.Collections.Generic;
using System.Linq;
using System.Net.Mail;
using System.Threading.Tasks;
using Microsoft.AspNetCore.Mvc;

namespace mywebapi.Controllers
{
    [Route("api/[controller]")]
    public class ValuesController : Controller
    {
        // GET api/values
        [HttpGet]
        public IEnumerable<string> Get()
        {
            return new string[] { "value1", "value2" };
        }

        // GET api/values/5
        [HttpGet("{id}")]
        public string Get(int id)
        {
            return "value";
        }

        // POST api/values
        [HttpPost]
        public void Post([FromBody]string value)
        {
        }

        // PUT api/values/5
        [HttpPut("{id}")]
        public void Put(int id, [FromBody]string value)
        {
        }

        // DELETE api/values/5
        [HttpDelete("{id}")]
        public void Delete(int id)
        {
        }

        [HttpPost("notify")]
        public IActionResult SendEmail(Message message)
        {
            MailMessage mailMessage = null;
            try
            {
                mailMessage = new MailMessage();
                MailAddress fromAddress = new MailAddress(message.From);
                mailMessage.From = fromAddress;
                mailMessage.To.Add(message.To);
                mailMessage.Body = message.Body;
                mailMessage.IsBodyHtml = true;
                mailMessage.Subject = message.Subject;
                SmtpClient smtpClient = new SmtpClient
                {
                    PickupDirectoryLocation = @"/home/eff/work/cs/mywebapi/email/",
                    Host = "localhost",
                    UseDefaultCredentials = true
                };
                smtpClient.Send(mailMessage);
            }
            catch (Exception ex)
            {

                return Ok("Failed: " + ex.ToString());
            }
            finally
            {
                mailMessage = null;
            }
            return Ok("Success");
        }

        public class Message
        {
            public string From { get; set; }
            public string To { get; set; }
            public string Body { get; set; }
            public string Subject { get; set; }
        }
    }
}
