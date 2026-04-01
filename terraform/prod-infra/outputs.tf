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