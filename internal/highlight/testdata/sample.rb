#!/usr/bin/env ruby

# This file uses tabs for indentation
# Ruby typically uses 2-space indentation

class Greeter
	def initialize(name)
		@name = name
	end

	def greet
		if @name && !@name.empty?
			puts "Hello, #{@name}!"
		else
			puts "Hello, World!"
		end
	end

	# Method with special characters: äöü ß 🚀
	def process_data(input)
		case input
		when String
			input.upcase
		when Integer
			input * 2
		else
			nil
		end
	end
end

# Test with multiple levels
def main
	greeter = Greeter.new("Ruby")
	greeter.greet

	# Array with tabs
	data = [
		"first	item",
		"second	item",
		"third	item"
	]

	data.each do |item|
		puts item
	end
end

main if __FILE__ == $PROGRAM_NAME
