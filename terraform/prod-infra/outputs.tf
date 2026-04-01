# outputs.tf

# Outputs are added here as we create each resource file.
# Example — after vpc.tf is created:
#
# output "vpc_id" {
#   description = "The VPC ID"
#   value       = aws_vpc.main.id
# }
output "vpc_id" {
    description = "The VPC ID"
    value = aws_vpc.main.id 
}

output "vpc_cidr" {
    description = "The VPC CIDR block"
    value = aws_vpc.main.cidr_block
}

output "public_subnet_ids" {
  description = "Map of public subnet name => ID"
  value       = { for k, v in aws_subnet.public : k => v.id }
}

output "private_subnet_ids" {
  description = "Map of private subnet name => ID"
  value       = { for k, v in aws_subnet.private : k => v.id }
}

output "isolated_subnet_ids" {
  description = "Map of isolated subnet name => ID"
  value       = { for k, v in aws_subnet.isolated : k => v.id }
}