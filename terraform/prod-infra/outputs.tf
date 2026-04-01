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

output "igw_id" {
  description = "Internet Gateway ID"
  value       = aws_internet_gateway.main.id
}

output "nat_gateway_ids" {
  description = "Map of AZ => NAT Gateway ID"
  value       = { for k, v in aws_nat_gateway.main : k => v.id }
}

output "public_route_table_id" {
  description = "Public route table ID"
  value       = aws_route_table.public.id
}

output "private_route_table_ids" {
  description = "Map of AZ => private route table ID"
  value       = { for k, v in aws_route_table.private : k => v.id }
}

output "alb_sg_id" {
  description = "ALB security group ID"
  value       = aws_security_group.alb.id
}

output "workers_sg_id" {
  description = "Workers security group ID"
  value       = aws_security_group.workers.id
}

output "db_sg_id" {
  description = "DB security group ID"
  value       = aws_security_group.db.id
}
